# Pipeline Run Log Integration Guide

## 概述

本指南介绍了新实现的流水线运行日志集成功能，该功能能够获取并显示阿里云DevOps流水线运行的完整日志信息，包括常规Job日志和VM部署日志。

## 新增功能

### 1. 获取流水线运行详情 (GetPipelineRunDetails)

基于阿里云官方API：[GetPipelineRun](https://help.aliyun.com/zh/yunxiao/developer-reference/getpipelinerun)

**功能**：
- 获取流水线运行的详细信息
- 包含所有阶段(Stage)和任务(Job)的列表
- 提供每个Job的ID、名称、状态、Actions等信息

**API签名**：
```go
func (c *Client) GetPipelineRunDetails(organizationId, pipelineId, pipelineRunId string) (*PipelineRunDetails, error)
```

### 2. 获取单个Job日志 (GetPipelineJobRunLog)

基于阿里云官方API：[GetPipelineJobRunLog](https://help.aliyun.com/zh/yunxiao/developer-reference/getpipelinejobrunlog)

**功能**：
- 获取指定Job的运行日志
- 支持实时日志获取
- 适用于常规构建、测试等类型的Job

**API签名**：
```go
func (c *Client) GetPipelineJobRunLog(organizationId, pipelineId, pipelineRunId, jobId string) (string, error)
```

### 3. 获取VM部署单详情 (GetVMDeployOrder)

基于阿里云官方API：[GetVMDeployOrder](https://help.aliyun.com/zh/yunxiao/developer-reference/getvmdeployorder)

**功能**：
- 获取VM部署单的详细信息
- 包含部署状态、批次信息、机器列表等
- 适用于包含GetVMDeployOrder action的部署类型Job

**API签名**：
```go
func (c *Client) GetVMDeployOrder(organizationId, pipelineId, deployOrderId string) (*VMDeployOrder, error)
```

### 4. 获取VM部署机器日志 (GetVMDeployMachineLog)

基于阿里云官方API：[GetVMDeployMachineLog](https://help.aliyun.com/zh/yunxiao/developer-reference/getvmdeploymachinelog)

**功能**：
- 获取指定机器的部署日志
- 包含部署开始/结束时间、日志内容等
- 支持批量获取多台机器的日志

**API签名**：
```go
func (c *Client) GetVMDeployMachineLog(organizationId, pipelineId, deployOrderId, machineSn string) (*VMDeployMachineLog, error)
```

### 5. 获取完整流水线日志 (GetPipelineRunLogs - 重构)

**功能**：
- 自动获取流水线运行中所有Job的日志
- 智能检测Job类型并使用相应的API获取日志
- 将各个Job的日志拼接成完整的日志视图
- 每个Job的日志前显示黄色标题，包含Job ID和名称
- 支持tview颜色格式化

**工作流程**：
1. 调用 `GetPipelineRunDetails` 获取Job列表和Actions信息
2. 遍历所有Stage和Job
3. 检测Job的Action类型：
   - 如果包含`GetVMDeployOrder` action：使用VM部署API获取日志
   - 否则：使用常规Job日志API
4. 格式化并拼接所有日志

## 数据结构

### JobAction 结构体
```go
type JobAction struct {
    Type        string                 `json:"type"`
    DisplayType string                 `json:"displayType"`
    Data        string                 `json:"data"`
    Disable     bool                   `json:"disable"`
    Params      map[string]interface{} `json:"params"`
    Name        string                 `json:"name"`
    Title       string                 `json:"title"`
    Order       interface{}            `json:"order"`
}
```

### Job 结构体 (更新)
```go
type Job struct {
    ID        int64       `json:"id"`
    JobSign   string      `json:"jobSign"`
    Name      string      `json:"name"`
    Status    string      `json:"status"`
    StartTime time.Time   `json:"startTime"`
    EndTime   time.Time   `json:"endTime"`
    Actions   []JobAction `json:"actions"`
    Result    string      `json:"result"`
}
```

### VMDeployOrder 结构体
```go
type VMDeployOrder struct {
    DeployOrderId     int                 `json:"deployOrderId"`
    Status            string              `json:"status"`
    Creator           string              `json:"creator"`
    CreateTime        int64               `json:"createTime"`
    UpdateTime        int64               `json:"updateTime"`
    CurrentBatch      int                 `json:"currentBatch"`
    TotalBatch        int                 `json:"totalBatch"`
    DeployMachineInfo VMDeployMachineInfo `json:"deployMachineInfo"`
}
```

### VMDeployMachine 结构体
```go
type VMDeployMachine struct {
    IP           string `json:"ip"`
    MachineSn    string `json:"machineSn"`
    Status       string `json:"status"`
    ClientStatus string `json:"clientStatus"`
    BatchNum     int    `json:"batchNum"`
    CreateTime   int64  `json:"createTime"`
    UpdateTime   int64  `json:"updateTime"`
}
```

### VMDeployMachineLog 结构体
```go
type VMDeployMachineLog struct {
    AliyunRegion    string `json:"aliyunRegion"`
    DeployBeginTime string `json:"deployBeginTime"`
    DeployEndTime   string `json:"deployEndTime"`
    DeployLog       string `json:"deployLog"`
    DeployLogPath   string `json:"deployLogPath"`
}
```

## 日志格式

### 常规Job日志格式
```
[yellow]Job #1: {jobName} (ID: {jobId})[-]
[yellow]Job Sign: {jobSign}[-]
[yellow]Status: {jobStatus}[-]
[yellow]Start Time: {startTime}[-]
[yellow]End Time: {endTime}[-]
[yellow]==================================================[-]
{actual job logs}
```

### VM部署Job日志格式
```
[yellow]Job #2: {jobName} (ID: {jobId})[-]
[yellow]Job Sign: {jobSign}[-]
[yellow]Status: {jobStatus}[-]
[yellow]Start Time: {startTime}[-]
[yellow]End Time: {endTime}[-]
[yellow]==================================================[-]
[yellow]Deploy Order ID: {deployOrderId}[-]
[yellow]Deploy Status: {deployStatus}[-]
[yellow]Current Batch: {currentBatch}/{totalBatch}[-]
[yellow]Host Group ID: {hostGroupId}[-]
[yellow]----------------------------------------[-]
[yellow]Machine #1: {machineIP} (SN: {machineSn})[-]
[yellow]Machine Status: {machineStatus}, Client Status: {clientStatus}[-]
[yellow]Batch: {batchNum}[-]
[yellow]..............................[-]
Deploy Begin Time: {deployBeginTime}
Deploy End Time: {deployEndTime}
Region: {aliyunRegion}
Log Path: {deployLogPath}
Deploy Log:
{actual deployment logs}
```

## 智能日志检测

系统会自动检测Job的Action类型：

1. **检测逻辑**：遍历Job的Actions数组，查找`type`为`GetVMDeployOrder`的action
2. **VM部署Job处理**：
   - 从Job的`result`字段中提取`deployOrderId`
   - 调用`GetVMDeployOrder`获取部署单详情
   - 遍历所有机器，调用`GetVMDeployMachineLog`获取每台机器的日志
3. **常规Job处理**：使用标准的`GetPipelineJobRunLog`API

## UI集成

在TUI界面中，当用户在运行历史中选择一个运行记录并按Enter键时：

1. 系统会调用新的 `GetPipelineRunLogs` 方法
2. 自动检测每个Job的类型并使用相应的API
3. 在日志视图中显示完整的格式化日志
4. Job标题和机器信息使用黄色高亮显示

## 测试

使用提供的测试文件 `test_log_integration.go` 来验证功能：

```bash
# 设置环境变量
export ALIYUN_DEVOPS_ENDPOINT="your-endpoint"
export ALIYUN_DEVOPS_TOKEN="your-token"
export ALIYUN_DEVOPS_ORG_ID="your-org-id"

# 编译并运行测试
go build -o test_log_integration test_log_integration.go
./test_log_integration
```

## 错误处理

- 如果无法获取运行详情，会返回相应错误
- 如果某个Job的日志获取失败，会在日志中显示错误信息，但不会中断其他Job的日志获取
- 如果VM部署单获取失败，会显示错误信息但继续处理其他Job
- 如果某台机器的日志获取失败，会显示错误信息但继续处理其他机器
- 如果Job没有日志，会显示"No logs available for this job"

## 性能考虑

- 日志获取是串行进行的，对于有大量Job和机器的流水线可能需要一些时间
- VM部署日志需要额外的API调用（每台机器一次）
- 每个Job和机器的日志都会完整加载到内存中
- 建议在生产环境中考虑添加超时和并发控制

## 兼容性

- 仅支持使用Personal Access Token的认证方式
- 需要阿里云DevOps API的相应权限
- 与现有的UI组件完全兼容
- 向后兼容：不包含VM部署action的Job仍使用原有的日志获取方式

## 未来改进

1. 添加并发日志获取以提高性能
2. 支持日志流式加载
3. 添加日志过滤和搜索功能
4. 支持日志导出功能
5. 添加更多Job类型的智能检测
6. 优化大量机器部署的日志显示 