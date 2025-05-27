# 日志状态栏集成和自动刷新优化

## 概述

对日志界面进行了重要的用户体验优化，将操作提示集成到状态栏中，并优化了自动刷新逻辑，确保只有需要的情况下才进行自动刷新。

## 主要改进

### 1. 状态栏集成操作提示

#### 修改前
- 日志内容底部显示固定的操作提示文字
- 所有情况下都显示 "Auto-refreshing every 5 seconds" 信息
- 提示信息占用日志显示空间

#### 修改后
- 将所有操作提示集成到底部状态栏中
- 状态栏动态显示不同信息：
  - 运行状态（带颜色标识）
  - 自动刷新状态（仅在需要时显示）
  - 操作提示（始终显示）

#### 状态栏格式
```
Status: [color]STATUS[-] | Auto-refresh: STATUS | Press 'r' to refresh, 'q' to return, 'e' to edit, 'v' to view in pager
```

### 2. 智能自动刷新逻辑

#### 新增全局变量
```go
isNewlyCreatedRun bool // 跟踪是否为新创建的运行
```

#### 自动刷新规则
1. **新创建的运行**: 始终自动刷新，直到完成后再刷新3次
2. **历史运行（RUNNING/QUEUED状态）**: 自动刷新，直到状态改变
3. **历史运行（已完成状态）**: 仅获取一次日志，不自动刷新

#### 状态栏显示逻辑
- **新创建的运行**: 显示自动刷新状态
- **正在运行的历史运行**: 显示自动刷新状态  
- **已完成的历史运行**: 不显示自动刷新信息

### 3. 用户体验改进

#### 清晰的视觉反馈
- **RUNNING**: 绿色状态文字
- **SUCCESS**: 白色状态文字
- **FAILED**: 红色状态文字
- **CANCELED**: 灰色状态文字

#### 智能提示信息
- 新创建运行: `Status: RUNNING | Auto-refresh: ON | Press 'r' to refresh...`
- 运行中历史: `Status: RUNNING | Auto-refresh: ON | Press 'r' to refresh...`
- 已完成历史: `Status: SUCCESS | Press 'r' to refresh, 'q' to return...`

#### 节省显示空间
- 移除日志内容底部的固定提示文字
- 状态栏紧凑显示所有必要信息
- 更多空间用于显示实际日志内容

## 技术实现

### 修改的函数

#### 1. `updateLogStatusBar()`
- 重构状态栏内容构建逻辑
- 根据运行类型动态显示自动刷新信息
- 集成操作提示到状态栏

#### 2. `startLogAutoRefresh()`
- 添加智能自动刷新判断逻辑
- 确保历史已完成运行不启动自动刷新
- 优化初始日志获取流程

#### 3. `runPipelineWithBranch()`
- 设置新创建运行标志 `isNewlyCreatedRun = true`

#### 4. 运行历史事件处理
- 设置历史运行标志 `isNewlyCreatedRun = false`
- 保持现有的状态判断逻辑

#### 5. `fetchAndDisplayLogs()`
- 移除底部固定提示文字
- 保留分隔线用于视觉分隔

### 代码变更摘要

```go
// 新增全局变量
isNewlyCreatedRun bool // 跟踪运行类型

// 状态栏内容构建
statusPart := fmt.Sprintf("Status: [color]%s[-]", currentRunStatus)
if isNewlyCreatedRun || strings.ToUpper(currentRunStatus) == "RUNNING" {
    autoRefreshPart = fmt.Sprintf(" | Auto-refresh: %s", getAutoRefreshStatus())
}
instructionsPart := " | Press 'r' to refresh, 'q' to return, 'e' to edit, 'v' to view in pager"

// 智能自动刷新
shouldAutoRefresh := isNewlyCreatedRun || strings.ToUpper(currentRunStatus) == "RUNNING"
if !shouldAutoRefresh {
    return // 不启动自动刷新
}
```

## 用户体验对比

### 修改前
```
Pipeline: my-pipeline
Run ID: 12345
...
[日志内容]
...
================================================================================
Auto-refreshing every 5 seconds. Press 'r' to refresh manually, 'q' to return, 'e' to edit in editor, 'v' to view in pager.

[状态栏] Status: RUNNING | Auto-refresh: ON
```

### 修改后
```
Pipeline: my-pipeline  
Run ID: 12345
...
[日志内容]
...
================================================================================

[状态栏] Status: RUNNING | Auto-refresh: ON | Press 'r' to refresh, 'q' to return, 'e' to edit, 'v' to view in pager
```

## 兼容性

- 完全向后兼容现有功能
- 不影响编辑器和分页器功能
- 保持所有现有键盘快捷键
- 状态栏信息更加丰富和智能

## 性能优化

- 减少不必要的自动刷新请求
- 历史已完成运行不消耗网络资源
- 更高效的状态栏更新逻辑
- 优化的日志显示空间利用

## 测试场景

1. **新创建运行**: 验证自动刷新和状态栏显示
2. **历史运行中**: 验证自动刷新和状态栏显示
3. **历史已完成**: 验证无自动刷新，状态栏不显示刷新信息
4. **手动刷新**: 验证 'r' 键功能正常
5. **编辑器/分页器**: 验证 'e'/'v' 键功能正常

## 总结

这次优化显著提升了日志界面的用户体验：
- 更清晰的状态信息显示
- 更智能的自动刷新逻辑
- 更高效的空间利用
- 更好的性能表现

用户现在可以清楚地知道当前运行的状态、是否在自动刷新，以及可用的操作选项，所有信息都集中在底部状态栏中，简洁而全面。 