package api

import (
	"encoding/json" // Added for dynamic parsing
	"fmt"
	"strconv" // Added for string to int conversion
	"strings" // Added for joining params
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests" // Added for requests.Integer
	"github.com/aliyun/alibaba-cloud-sdk-go/services/devops-rdc" // Changed import path
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/auth/credentials"
)

// Pipeline represents a pipeline in Aliyun DevOps.
type Pipeline struct {
	PipelineID    string    `json:"pipelineId"`
	Name          string    `json:"name"`
	Status        string    `json:"status"`        // e.g., RUNNING, SUCCESS, FAILED, CANCELED
	LastRunStatus string    `json:"lastRunStatus"` // Status of the most recent run
	Creator       string    `json:"creator"`
	Modifier      string    `json:"modifier"`
	CreateTime    time.Time `json:"createTime"`
	UpdateTime    time.Time `json:"updateTime"`
}

// PipelineRun represents a single execution of a pipeline.
type PipelineRun struct {
	RunID       string    `json:"runId"`
	PipelineID  string    `json:"pipelineId"`
	Status      string    `json:"status"`     // e.g., RUNNING, SUCCESS, FAILED, CANCELED
	StartTime   time.Time `json:"startTime"`
	FinishTime  time.Time `json:"finishTime"`
	TriggerMode string    `json:"triggerMode"` // e.g., MANUAL, PUSH, SCHEDULE
}

// PipelineGroup represents a group of pipelines.
type PipelineGroup struct {
	GroupID string `json:"groupId"`
	Name    string `json:"name"`
}

// Client is a client for interacting with the Aliyun DevOps API.
type Client struct {
	sdkClient *devops_rdc.Client // Changed to devops_rdc
}

// NewClient creates a new Aliyun DevOps API client.
// If regionId is empty, "cn-hangzhou" will be used.
func NewClient(accessKeyId, accessKeySecret, regionId string) (*Client, error) {
	if regionId == "" {
		regionId = "cn-hangzhou" // Default region
	}

	credential := credentials.NewAccessKeyCredential(accessKeyId, accessKeySecret)
	sdkClient, err := devops_rdc.NewClientWithOptions(regionId, sdk.NewConfig(), credential) // Changed to devops_rdc
	if err != nil {
		return nil, fmt.Errorf("failed to create devops-rdc client: %w", err)
	}

	return &Client{sdkClient: sdkClient}, nil
}

// ListPipelines retrieves a list of pipelines for a given organization.
func (c *Client) ListPipelines(organizationId string) ([]Pipeline, error) {
	if organizationId == "" {
		return nil, fmt.Errorf("organizationId is required for ListPipelines")
	}
	request := devops_rdc.CreateListPipelinesRequest() // Changed to devops_rdc
	request.Scheme = "https" // Usually HTTPS
	request.OrgId = organizationId // Assuming OrgId based on typical SDK patterns for devops-rdc

	// TODO: Add pagination handling if the API supports it.
	// request.NextToken / request.MaxResults might be relevant for pagination.

	response, err := c.sdkClient.ListPipelines(request)
	if err != nil {
		return nil, fmt.Errorf("failed to list pipelines: %w", err)
	}

	var pipelines []Pipeline
	// Assuming response.Pipelines is a slice of the SDK's pipeline type.
	// The actual field name might be different, e.g., `response.Data.Pipelines` or `response.Object`.
	// This needs verification against the actual SDK structure.
	// For now, we'll try to access `response.Pipelines` directly.
	// If `response.Pipelines` is nil or not the correct path, this will fail at runtime or not map data.
	
	// A common structure for Aliyun SDK responses is a `RequestId` field and then a data field.
	// Let's assume the list of pipelines is in `response.Pipelines`.
	// The type of `p` here is `*devops_rdc.ListPipelinesPipelines`.
	// We need to check the fields of this struct.
	// Example: p.PipelineId (int64), p.Name (string), p.Status (string),
	// p.Creator (string), p.Modifier (string), p.GmtCreate (string), p.GmtModified (string)

	// According to the SDK source for list_pipelines.go:
	// type ListPipelinesResponse struct {
	//         *responses.BaseResponse
	//         Success      bool                   `json:"Success" xml:"Success"`
	//         ErrorCode    string                 `json:"ErrorCode" xml:"ErrorCode"`
	//         ErrorMessage string                 `json:"ErrorMessage" xml:"ErrorMessage"`
	//         Object       map[string]interface{} `json:"Object" xml:"Object"`  // THIS IS THE KEY FIELD
	//         RequestId    string                 `json:"RequestId" xml:"RequestId"`
	// }
	// The actual pipelines are likely inside response.Object, e.g., response.Object["Pipelines"]

	if !response.Success {
		return nil, fmt.Errorf("API error: %s (ErrorCode: %s)", response.ErrorMessage, response.ErrorCode)
	}

	if response.Object == nil {
		return []Pipeline{}, nil // No data, but request was successful
	}
	
	// Try to get the list of pipelines from response.Object. Common key names are "Pipelines", "List", "Items".
	// Let's assume "Pipelines" is the key.
	rawPipelines, ok := response.Object["Pipelines"]
	if !ok {
		return nil, fmt.Errorf("field 'Pipelines' not found in response.Object. Available keys: %v", getMapKeys(response.Object))
	}

	pipelineItems, ok := rawPipelines.([]interface{})
	if !ok {
		return nil, fmt.Errorf("'Pipelines' field in response.Object is not a slice (actual type: %T)", rawPipelines)
	}

	for _, item := range pipelineItems {
		sdkPipeline, ok := item.(map[string]interface{})
		if !ok {
			// Log or skip malformed item
			continue
		}

		var createTime, updateTime time.Time
		// Field names are based on `ListPipelinesPipelines` struct from SDK:
		// Name, PipelineId, Status, CreateTime, UpdateTime, Creator, Modifier.
		// CreateTime, UpdateTime are int64 milliseconds.

		if ct, ok := sdkPipeline["CreateTime"].(float64); ok && ct > 0 { // JSON numbers often decode to float64
			createTime = time.Unix(int64(ct)/1000, 0)
		}
		if ut, ok := sdkPipeline["UpdateTime"].(float64); ok && ut > 0 {
			updateTime = time.Unix(int64(ut)/1000, 0)
		}
		
		pipelineIdFloat, _ := sdkPipeline["PipelineId"].(float64)


		pipe := Pipeline{
			PipelineID:    fmt.Sprintf("%d", int64(pipelineIdFloat)),
			Name:          getStringField(sdkPipeline, "Name"),
			Status:        getStringField(sdkPipeline, "Status"),
			Creator:       getStringField(sdkPipeline, "Creator"),
			Modifier:      getStringField(sdkPipeline, "Modifier"),
			CreateTime:    createTime,
			UpdateTime:    updateTime,
			// LastRunStatus is not in this response, will need another call or is part of GetPipelineDetails
		}
		pipelines = append(pipelines, pipe)
	}

	return pipelines, nil
}

// Helper function to safely get string fields from map[string]interface{}
func getStringField(data map[string]interface{}, key string) string {
	if val, ok := data[key].(string); ok {
		return val
	}
	return ""
}

// Helper function to get keys from a map for error reporting
func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// GetPipelineDetails retrieves details for a specific pipeline.
func (c *Client) GetPipelineDetails(organizationId string, pipelineId string) (*Pipeline, error) {
	// request := devops_rdc.CreateGetPipelineRequest() // Or similar
	// request.OrgId = organizationId
	// request.PipelineId = pipelineId
	// ...
	return nil, fmt.Errorf("not implemented: GetPipelineDetails")
}

// RunPipeline triggers a pipeline run using the ExecutePipeline SDK method.
func (c *Client) RunPipeline(organizationId string, pipelineIdStr string, params map[string]string) (*PipelineRun, error) {
	if organizationId == "" {
		return nil, fmt.Errorf("organizationId is required")
	}
	if pipelineIdStr == "" {
		return nil, fmt.Errorf("pipelineId is required")
	}

	pipelineIdInt, err := strconv.ParseInt(pipelineIdStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pipelineId to int64: %w", err)
	}

	request := devops_rdc.CreateExecutePipelineRequest()
	request.Scheme = "https"
	request.OrgId = organizationId
	request.PipelineId = requests.NewInteger(pipelineIdInt)

	// Convert params map to "key1=value1,key2=value2" string format
	// TODO: Confirm the exact format required by the Aliyun API for Parameters.
	// Assuming "key1=value1,key2=value2" or JSON string. For now, using the former.
	var paramList []string
	for key, value := range params {
		paramList = append(paramList, fmt.Sprintf("%s=%s", key, value))
	}
	request.Parameters = strings.Join(paramList, ",") // Example format

	response, err := c.sdkClient.ExecutePipeline(request)
	if err != nil {
		return nil, fmt.Errorf("failed to execute pipeline: %w", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("API error executing pipeline: %s (ErrorCode: %s)", response.ErrorMessage, response.ErrorCode)
	}

	// The response.Object is the RunId (int64)
	runId := response.Object

	// The ExecutePipeline API only returns the RunID.
	// To return a full PipelineRun struct, we should ideally call GetPipelineRun here.
	// However, to keep this method focused, we'll return a minimal PipelineRun object.
	// The caller can then use GetPipelineRun to fetch full details if needed.
	return &PipelineRun{
		RunID:      fmt.Sprintf("%d", runId),
		PipelineID: pipelineIdStr,
		Status:     "QUEUED", // Assuming it's queued; actual status needs GetPipelineRun
		// StartTime would be set by GetPipelineRun
	}, nil
}

// StopPipelineRun stops a pipeline run.
func (c *Client) StopPipelineRun(organizationId string, pipelineId string, runId string) error {
	// request := devops_rdc.CreateStopPipelineRunRequest() // Or similar
	// request.OrgId = organizationId
	// request.PipelineId = pipelineId
	// request.RunId = runId
	// ...
	return fmt.Errorf("not implemented: StopPipelineRun")
}

// GetPipelineRun retrieves details of a specific pipeline run using GetPipelineInstanceInfo SDK method.
func (c *Client) GetPipelineRun(organizationId string, pipelineIdStr string, runIdStr string) (*PipelineRun, error) {
	if organizationId == "" {
		return nil, fmt.Errorf("organizationId is required")
	}
	if pipelineIdStr == "" {
		return nil, fmt.Errorf("pipelineId is required")
	}
	if runIdStr == "" {
		return nil, fmt.Errorf("runId is required")
	}

	pipelineIdInt, err := strconv.ParseInt(pipelineIdStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pipelineId to int64: %w", err)
	}

	request := devops_rdc.CreateGetPipelineInstanceInfoRequest()
	request.Scheme = "https"
	request.OrgId = organizationId
	request.PipelineId = requests.NewInteger(pipelineIdInt)
	request.FlowInstanceId = runIdStr // FlowInstanceId is the RunId

	response, err := c.sdkClient.GetPipelineInstanceInfo(request)
	if err != nil {
		return nil, fmt.Errorf("failed to get pipeline instance info: %w", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("API error getting pipeline instance info: %s (ErrorCode: %s)", response.ErrorMessage, response.ErrorCode)
	}

	// response.Object is of type devops_rdc.Object (which is an alias for struct{} in the generated code if not defined)
	// Need to parse this dynamically, similar to ListPipelineGroups
	dataBytes, err := json.Marshal(response.Object)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response.Object for GetPipelineRun: %w", err)
	}

	var runMap map[string]interface{}
	if err := json.Unmarshal(dataBytes, &runMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response.Object into map for GetPipelineRun: %w. Raw JSON: %s", err, string(dataBytes))
	}

	// Extract fields from runMap
	// Common fields from a pipeline run instance:
	// Status, StartTime, FinishTime, TriggerMode, RunId (usually FlowInstanceId itself), PipelineId
	// Timestamps might be in milliseconds (float64) or string format.
	// Status might be "SUCCESS", "RUNNING", "FAILED", "CANCELED", "QUEUED" etc.

	var startTime, finishTime time.Time
	if st, ok := runMap["StartTime"].(float64); ok { // Assuming milliseconds
		startTime = time.Unix(int64(st)/1000, 0)
	} else if stStr, ok := runMap["StartTime"].(string); ok {
		startTime, _ = time.Parse(time.RFC3339Nano, stStr) // Or other common time formats
	}
	// Similar for FinishTime
	if ft, ok := runMap["FinishTime"].(float64); ok {
		finishTime = time.Unix(int64(ft)/1000, 0)
	} else if ftStr, ok := runMap["FinishTime"].(string); ok {
		finishTime, _ = time.Parse(time.RFC3339Nano, ftStr)
	}
	
	// Assuming `Id` or `FlowInstanceId` for RunID from the map.
	// `PipelineId` should also be present.
	// `Status` and `TriggerMode` are important.

	pipelineRun := &PipelineRun{
		RunID:       getStringField(runMap, "Id"), // Or "FlowInstanceId", "InstanceId"
		PipelineID:  fmt.Sprintf("%d", int64(getNumberField(runMap, "PipelineId"))), // Assuming "PipelineId" is a number
		Status:      getStringField(runMap, "Status"),
		TriggerMode: getStringField(runMap, "TriggerMode"), // Or "triggerMode", "triggerType"
		StartTime:   startTime,
		FinishTime:  finishTime,
	}
	
	// If RunID was not found by "Id", try "FlowInstanceId"
	if pipelineRun.RunID == "" {
		pipelineRun.RunID = getStringField(runMap, "FlowInstanceId")
	}
    if pipelineRun.RunID == "" { // Fallback to the input runId if not found in response
        pipelineRun.RunID = runIdStr
    }
    if pipelineRun.PipelineID == "0" || pipelineRun.PipelineID == "" { // Fallback for PipelineID
        pipelineRun.PipelineID = pipelineIdStr
    }


	// The subtask requires mapping Status correctly.
	// Statuses like "SUCCESS", "FAILED", "RUNNING", "CANCELED", "WAITING", "QUEUED" are common.
	// The `getStringField(runMap, "Status")` should capture this if the key "Status" is correct.

	return pipelineRun, nil
}

// GetPipelineRunLogs retrieves logs for a specific job within a pipeline run.
// Note: Aliyun DevOps RDC API (GetPipelineLog) fetches logs per JobId, not directly per RunId.
// This implementation currently requires a JobId and returns an error if it's not provided,
// as determining the JobId from a RunId is not yet implemented.
func (c *Client) GetPipelineRunLogs(organizationId string, pipelineIdStr string, runIdStr string /* TODO: Add jobId string */) (string, error) {
	// To fully implement this, we would need:
	// 1. A way to list jobs for a given runId (e.g., from GetPipelineInstanceInfo or a new ListJobsInPipelineRun API).
	// 2. Then, for each job (or a selected one), call GetPipelineLog.
	// For now, this function will be a placeholder requiring JobId if we were to call GetPipelineLog.

	// The SDK method is `GetPipelineLog(request *GetPipelineLogRequest)`
	// It requires `request.JobId` (requests.Integer) and `request.PipelineId` (requests.Integer).
	// `request.OrgId` is also required.
	// The response `GetPipelineLogResponse.Object` is `[]Job`. Each `Job` struct would contain log segments.
	// The internal structure of `Job` (e.g., how logs are stored) is not in the viewed `get_pipeline_log.go`.

	return "", fmt.Errorf("not implemented: GetPipelineRunLogs. This API requires a JobId. Please first implement listing jobs for a run to get a JobId, then call GetPipelineLog for that JobId. RunId was: %s, PipelineId: %s", runIdStr, pipelineIdStr)

	// Example structure if JobId was known:
	/*
	    if organizationId == "" || pipelineIdStr == "" || jobIdStr == "" {
	        return "", fmt.Errorf("organizationId, pipelineId, and jobId are required for GetPipelineRunLogs")
	    }
	    pipelineIdInt, err := strconv.ParseInt(pipelineIdStr, 10, 64)
	    if err != nil { return "", fmt.Errorf("invalid pipelineId: %w", err) }
	    jobIdInt, err := strconv.ParseInt(jobIdStr, 10, 64)
	    if err != nil { return "", fmt.Errorf("invalid jobId: %w", err) }

	    request := devops_rdc.CreateGetPipelineLogRequest()
	    request.Scheme = "https"
	    request.OrgId = organizationId
	    request.PipelineId = requests.NewInteger(pipelineIdInt)
	    request.JobId = requests.NewInteger(jobIdInt)

	    response, err := c.sdkClient.GetPipelineLog(request)
	    if err != nil {
	        return "", fmt.Errorf("failed to get pipeline log: %w", err)
	    }
	    if !response.Success {
	        return "", fmt.Errorf("API error getting pipeline log: %s (ErrorCode: %s)", response.ErrorMessage, response.ErrorCode)
	    }

	    // response.Object is []devops_rdc.Job. The structure of Job is not in get_pipeline_log.go.
	    // Assuming each Job object has a field like "LogContent" (string) or "LogEntries" ([]string).
	    var allLogs strings.Builder
	    dataBytes, err := json.Marshal(response.Object)
	    if err != nil {
	        return "", fmt.Errorf("failed to marshal log response.Object: %w", err)
	    }
	    var jobsData []map[string]interface{}
	    if err := json.Unmarshal(dataBytes, &jobsData); err != nil {
	        return "", fmt.Errorf("failed to unmarshal log response.Object into jobsData: %w", err)
	    }

	    for _, jobMap := range jobsData {
	        // Assuming each jobMap contains log information.
	        // Need to find the key for log content, e.g., "Content", "Log", "Steps" then their logs.
	        if logContent, ok := jobMap["LogContent"].(string); ok { // This key "LogContent" is a guess.
	            allLogs.WriteString(logContent)
	            allLogs.WriteString("\n")
	        }
	        // If logs are per step within a job, more complex parsing is needed.
	    }
	    if allLogs.Len() == 0 && len(jobsData) > 0 {
	        return "", fmt.Errorf("logs found for job %s, but content parsing failed. Raw job data: %+v", jobIdStr, jobsData[0])
	    }
	    return allLogs.String(), nil
	*/
}

// getNumberField is a helper for dynamic map parsing
func getNumberField(data map[string]interface{}, key string) float64 {
	if val, ok := data[key].(float64); ok { // JSON numbers are often float64
		return val
	}
	// Could add more type checks if needed (e.g., string to float64)
	return 0
}

// ListPipelineRuns retrieves a list of runs for a specific pipeline.

// ListPipelineRuns retrieves a list of runs for a specific pipeline.
func (c *Client) ListPipelineRuns(organizationId string, pipelineId string) ([]PipelineRun, error) {
	// request := devops_rdc.CreateListPipelineRunsRequest()
	// request.OrgId = organizationId
	// request.PipelineId = pipelineId
	// ...
	return nil, fmt.Errorf("not implemented: ListPipelineRuns")
}

// ListPipelineGroups retrieves a list of pipeline groups (projects) for an organization.
func (c *Client) ListPipelineGroups(organizationId string) ([]PipelineGroup, error) {
	if organizationId == "" {
		return nil, fmt.Errorf("organizationId is required for ListPipelineGroups")
	}

	request := devops_rdc.CreateListDevopsProjectsRequest()
	request.Scheme = "https"
	request.OrgId = organizationId
	// request.PageSize = "100" // Example: Add pagination if needed and supported

	response, err := c.sdkClient.ListDevopsProjects(request)
	if err != nil {
		return nil, fmt.Errorf("failed to list devops projects (pipeline groups): %w", err)
	}

	if !response.Successful { // Note: Field name is `Successful` not `Success`
		return nil, fmt.Errorf("API error listing devops projects: %s (ErrorCode: %s)", response.ErrorMsg, response.ErrorCode)
	}

	var groups []PipelineGroup

	// The response.Object is of type devops_rdc.ObjectInListDevopsProjects
	// We need to determine its structure. Assuming it has a field like 'Result' or 'Projects'
	// which is a slice of project details.
	// For now, let's try to access a field named 'Result' and expect it to be []interface{}.
	// This is a common pattern. If this fails, we'll need to find the actual field name.
	// The Go SDK doesn't provide the nested struct definitions directly in the viewed file for `ObjectInListDevopsProjects`.
	// We'll have to rely on reflection or common patterns.
	// A safer way if the exact structure is unknown is to marshal response.Object to JSON and then unmarshal to a known map.
	// However, let's try direct access with an assumption.
	// Assume response.Object has a field `Result` which is a slice of maps.
	// This part is speculative due to missing definition of ObjectInListDevopsProjects.
	//
	// If `response.Object` itself is the list or contains the list directly, this will need adjustment.
	// The SDK source for `list_devops_projects.go` defines:
	// type ListDevopsProjectsResponse struct { ... Object ObjectInListDevopsProjects ... }
	// The type `ObjectInListDevopsProjects` is not defined in that specific file.
	//
	// Let's assume `ObjectInListDevopsProjects` has a field `Projects` (a common name)
	// or perhaps `Result`. Trying `Projects` first.
	// If `response.Object.Projects` exists and is a slice:
	// We cannot directly access `response.Object.Projects` without knowing the type `ObjectInListDevopsProjects`.
	//
	// A common way SDKs handle this is that `response.Object` might be a struct, and `response.Object.Projects` is the field.
	// Or `response.Object` is a map itself.
	// Given `Object       map[string]interface{} `json:"Object" xml:"Object"` in ListPipelinesResponse,
	// and `Object     ObjectInListDevopsProjects `json:"Object" xml:"Object"` in ListDevopsProjectsResponse,
	// it's likely ObjectInListDevopsProjects is a struct, not a map.
	//
	// The SDK code generator usually creates structs for these.
	// Let's try to access common field names like "Projects" or "Items" or "List" from response.Object
	// This requires knowing the structure of `ObjectInListDevopsProjects`.
	//
	// If we assume it's similar to ListPipelines and the actual data is in a map *within* Object,
	// but `Object` here is a specific struct type, not `map[string]interface{}`.
	// This means `Object` itself should have fields.
	// Let's try to find the definition of `ObjectInListDevopsProjects` by searching the SDK repo or making an educated guess.
	// A common structure is `TotalCount` and `Items []ActualItem`.
	//
	// Given the subtask asks to report on the structure, if this fails, the error will be informative.
	// For now, to make it compilable, I will assume `response.Object` has a field `Result` which is a slice.
	// This will likely fail at runtime if `Result` is not the field, or not a slice.
	// The Go way would be to check the actual generated SDK code.
	//
	// A more robust approach without full introspection tools here:
	// Assume `response.Object` has a field `Projects` which is a slice of structs,
	// and each struct has `ProjectId` and `Name`.
	// This still requires knowing the exact field name (`Projects`) and struct fields.
	//
	// Let's try to get the raw JSON of `response.Object` and unmarshal it.
	// This is a workaround for not having the exact struct definition.
	// Note: This is not ideal but can work if the structure is simple JSON.

	// For now, returning an error, as the structure of `ObjectInListDevopsProjects` is unknown.
	// The next step would be to inspect the SDK's generated code for this type.
	// If `ObjectInListDevopsProjects` has a field like `Items` or `Projects` of type `[]struct { ProjectId string; Name string; ...}`
	// then that would be used.
	// Example (pseudo-code, assuming `response.Object.Result` is `[]map[string]interface{}` which is a common dynamic way):
	/*
	    dataBytes, err := json.Marshal(response.Object)
	    if err != nil {
	        return nil, fmt.Errorf("failed to marshal response.Object: %w", err)
	    }

	    var tempObj struct {
	        // Try common field names for lists. "Result", "Items", "Projects", "List"
	        Result []map[string]interface{} `json:"Result"` // Or "Projects", "Items", etc.
	    }
	    if err := json.Unmarshal(dataBytes, &tempObj); err != nil {
	        return nil, fmt.Errorf("failed to unmarshal response.Object into tempObj: %w", err)
	    }

	    if tempObj.Result == nil {
	        // Try another key if Result was not found, e.g. "Projects"
	        // This indicates the assumed key "Result" was wrong.
	        // For now, we'll assume "Result" or fail.
	         return nil, fmt.Errorf("no 'Result' field found in unmarshalled response.Object, or it's null. Raw Object: %s", string(dataBytes))
	    }

	    for _, itemMap := range tempObj.Result {
	        groupID := getStringField(itemMap, "ProjectId") // Or "Id"
	        name := getStringField(itemMap, "Name")
	        if groupID != "" && name != "" {
	            groups = append(groups, PipelineGroup{
	                GroupID: groupID,
	                Name:    name,
	            })
	        }
	    }
	*/
	// The above JSON marshalling/unmarshalling is a robust way to explore unknown structs.
	// However, to proceed with the subtask, I need to make a direct assumption or state that it's blocked by unknown struct.
	// The subtask asks to report on the structure. The current structure is `devops_rdc.ObjectInListDevopsProjects`.
	// The fields within this struct are unknown.

	// Let's assume, based on common SDK patterns, that `ObjectInListDevopsProjects` might have a field named `Projects`.
	// And this field is a slice of structs, each having `ProjectId` and `Name`.
	// This is a strong assumption.
	// To make this compile, I would need the definition of `ObjectInListDevopsProjects`.
	// Since I don't have it, I cannot write the direct field access code that compiles.

	// Reporting: The API call used is `ListDevopsProjects`.
	// The response structure for the list of projects is within `response.Object` of type `devops_rdc.ObjectInListDevopsProjects`.
	// The internal structure of `devops_rdc.ObjectInListDevopsProjects` is not defined in the viewed SDK files.
	// To handle this, we marshal `response.Object` to JSON and then unmarshal it into a map[string]interface{}
	// to dynamically inspect its fields.

	dataBytes, err := json.Marshal(response.Object)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response.Object for dynamic parsing: %w. API Response: Successful=%v, ErrorCode=%s, ErrorMsg=%s", err, response.Successful, response.ErrorCode, response.ErrorMsg)
	}

	var objectMap map[string]interface{}
	if err := json.Unmarshal(dataBytes, &objectMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response.Object into map: %w. Raw Object JSON: %s", err, string(dataBytes))
	}

	// Now, try to find the list of projects within objectMap.
	// Common keys for lists: "Projects", "Items", "List", "Result", "Data"
	possibleKeys := []string{"Projects", "Result", "Items", "List", "Data"}
	var projectListInterfaces []interface{}
	foundListKey := ""

	for _, key := range possibleKeys {
		if listData, ok := objectMap[key].([]interface{}); ok {
			projectListInterfaces = listData
			foundListKey = key
			break
		}
	}

	if projectListInterfaces == nil {
		return nil, fmt.Errorf("could not find a known key ('%v') for project list in response.Object map. Available keys: %v. Raw Object JSON: %s", possibleKeys, getMapKeys(objectMap), string(dataBytes))
	}
	
	if len(projectListInterfaces) == 0 {
		return groups, nil // No projects found, but the call was successful
	}

	for _, projectInterface := range projectListInterfaces {
		projectMap, ok := projectInterface.(map[string]interface{})
		if !ok {
			// Log or skip malformed item
			// fmt.Fprintf(os.Stderr, "Warning: project item is not a map[string]interface{}: %T\n", projectInterface)
			continue
		}

		var group PipelineGroup
		// Try common keys for ID: "ProjectId", "Id", "ID"
		// Try common keys for Name: "Name", "ProjectName"
		
		if idVal, ok := projectMap["ProjectId"].(string); ok {
			group.GroupID = idVal
		} else if idVal, ok := projectMap["Id"].(string); ok {
			group.GroupID = idVal
		} else if idVal, ok := projectMap["ID"].(string); ok {
			group.GroupID = idVal
		} else if idValFloat, ok := projectMap["ProjectId"].(float64); ok { // Sometimes numbers come as float64
            group.GroupID = fmt.Sprintf("%.0f", idValFloat)
        } else if idValFloat, ok := projectMap["Id"].(float64); ok {
             group.GroupID = fmt.Sprintf("%.0f", idValFloat)
        } else if idValFloat, ok := projectMap["ID"].(float64); ok {
             group.GroupID = fmt.Sprintf("%.0f", idValFloat)
        }


		if nameVal, ok := projectMap["Name"].(string); ok {
			group.Name = nameVal
		} else if nameVal, ok := projectMap["ProjectName"].(string); ok {
			group.Name = nameVal
		}

		if group.GroupID != "" && group.Name != "" {
			groups = append(groups, group)
		} else {
			// Could log a warning if a project-like map was found but key fields were missing/empty
			// fmt.Fprintf(os.Stderr, "Warning: Found project map but GroupID or Name is missing/empty: %+v\n", projectMap)
		}
	}
	
	if len(groups) == 0 && len(projectListInterfaces) > 0 {
		// This means we found project items, but couldn't extract ID/Name from any of them.
		return nil, fmt.Errorf("found %d project items under key '%s', but failed to extract GroupID/Name from any. Check API response structure and expected keys. First item: %+v", len(projectListInterfaces), foundListKey, projectListInterfaces[0])
	}

	return groups, nil
}
