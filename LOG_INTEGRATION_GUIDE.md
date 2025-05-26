# Pipeline Run Log Integration Guide

## 概述

本指南介绍了新实现的流水线运行日志集成功能，该功能能够获取并显示阿里云DevOps流水线运行的完整日志信息。

## 新增功能

### 1. 获取流水线运行详情 (GetPipelineRunDetails)

基于阿里云官方API：[GetPipelineRun](https://help.aliyun.com/zh/yunxiao/developer-reference/getpipelinerun)

**功能**：
- 获取流水线运行的详细信息
- 包含所有阶段(Stage)和任务(Job)的列表
- 提供每个Job的ID、名称、状态等信息

**API签名**：
```go
func (c *Client) GetPipelineRunDetails(organizationId, pipelineId, pipelineRunId string) (*PipelineRunDetails, error)
```

### 2. 获取单个Job日志 (GetPipelineJobRunLog)

基于阿里云官方API：[GetPipelineJobRunLog](https://help.aliyun.com/zh/yunxiao/developer-reference/getpipelinejobrunlog)

**功能**：
- 获取指定Job的运行日志
- 支持实时日志获取

**API签名**：
```go
func (c *Client) GetPipelineJobRunLog(organizationId, pipelineId, pipelineRunId, jobId string) (string, error)
```

### 3. 获取完整流水线日志 (GetPipelineRunLogs - 重构)

**功能**：
- 自动获取流水线运行中所有Job的日志
- 将各个Job的日志拼接成完整的日志视图
- 每个Job的日志前显示黄色标题，包含Job ID和名称
- 支持tview颜色格式化

**工作流程**：
1. 调用 `GetPipelineRunDetails` 获取Job列表
2. 遍历所有Stage和Job
3. 对每个Job调用 `GetPipelineJobRunLog` 获取日志
4. 格式化并拼接所有日志

## 数据结构

### Job 结构体
```go
type Job struct {
    ID       int64     `json:"id"`
    JobSign  string    `json:"jobSign"`
    Name     string    `json:"name"`
    Status   string    `json:"status"`
    StartTime time.Time `json:"startTime"`
    EndTime   time.Time `json:"endTime"`
}
```

### Stage 结构体
```go
type Stage struct {
    Index string `json:"index"`
    Name  string `json:"name"`
    Jobs  []Job  `json:"jobs"`
}
```

### PipelineRunDetails 结构体
```go
type PipelineRunDetails struct {
    PipelineRunID int64   `json:"pipelineRunId"`
    PipelineID    int64   `json:"pipelineId"`
    Status        string  `json:"status"`
    TriggerMode   int     `json:"triggerMode"`
    CreateTime    int64   `json:"createTime"`
    UpdateTime    int64   `json:"updateTime"`
    Stages        []Stage `json:"stages"`
}
```

## 日志格式

新的日志格式包含以下信息：

```
Pipeline Run Logs - Run ID: {runId}
Pipeline ID: {pipelineId}
Status: {status}
================================================================================

[yellow]Stage: {stageName} ({stageIndex})[-]
------------------------------------------------------------

[yellow]Job #1: {jobName} (ID: {jobId})[-]
[yellow]Job Sign: {jobSign}[-]
[yellow]Status: {jobStatus}[-]
[yellow]Start Time: {startTime}[-]
[yellow]End Time: {endTime}[-]
[yellow]==================================================[-]
{actual job logs}

================================================================================

Total jobs processed: {jobCount}
```

## UI集成

在TUI界面中，当用户在运行历史中选择一个运行记录并按Enter键时：

1. 系统会调用新的 `GetPipelineRunLogs` 方法
2. 自动获取所有Job的日志并拼接
3. 在日志视图中显示完整的格式化日志
4. Job标题使用黄色高亮显示

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
- 如果Job没有日志，会显示"No logs available for this job"

## 性能考虑

- 日志获取是串行进行的，对于有大量Job的流水线可能需要一些时间
- 每个Job的日志都会完整加载到内存中
- 建议在生产环境中考虑添加超时和并发控制

## 兼容性

- 仅支持使用Personal Access Token的认证方式
- 需要阿里云DevOps API的相应权限
- 与现有的UI组件完全兼容

## 未来改进

1. 添加并发日志获取以提高性能
2. 支持日志流式加载
3. 添加日志过滤和搜索功能
4. 支持日志导出功能 