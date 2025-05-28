# Vim风格搜索实现

## 功能概述

在日志查看界面实现了vim风格的搜索功能，支持：

- 按 `/` 键进入搜索模式
- 实时搜索和高亮显示匹配项
- 使用 `n` 和 `N` 键在匹配项之间导航
- 按 `Esc` 键退出搜索模式

## 实现细节

### 搜索流程

1. **进入搜索模式**：
   - 按 `/` 键触发 `startLogSearch()` 函数
   - 创建搜索输入框并添加到日志页面布局
   - 将焦点设置到搜索输入框

2. **实时搜索**：
   - 用户输入时触发 `performLogSearch()` 函数
   - 执行大小写不敏感的搜索
   - 高亮显示所有匹配项，当前匹配项用金色背景标识

3. **确认搜索**：
   - 按回车键后将焦点转移到日志文本框
   - 这样用户可以立即使用 `n`/`N` 快捷键导航

4. **导航匹配项**：
   - `n` 键：跳转到下一个匹配项
   - `N` 键：跳转到上一个匹配项
   - 自动滚动到匹配项位置

5. **退出搜索**：
   - 按 `Esc` 键退出搜索模式
   - 清空搜索输入框内容
   - 恢复原始日志文本显示
   - 恢复正常的日志页面布局

### 关键修复

1. **焦点管理**：
   - 修复了回车后焦点仍在搜索框的问题
   - 现在回车后自动将焦点转移到日志文本框，确保 `n`/`N` 快捷键可用

2. **状态清理**：
   - 修复了退出搜索时搜索框内容未清空的问题
   - 现在退出搜索时会自动清空搜索输入框

### 高亮显示

- 当前匹配项：金色前景 + 灰色背景 `[gold:gray]`
- 其他匹配项：白色前景 + 灰色背景 `[white:gray]`
- 状态栏显示匹配数量和当前位置

### 状态栏信息

搜索模式下状态栏显示：
- 有匹配项：`Search: 'keyword' (1/5 matches) | 'n' next, 'N' prev, Esc to exit`
- 无匹配项：`Search: 'keyword' (no matches) | Esc to exit`
- 搜索模式：`Search mode | Enter search term, Esc to exit`

## 使用方法

1. 在日志查看界面按 `/` 进入搜索模式
2. 输入搜索关键词（支持实时搜索）
3. 按回车确认搜索，焦点自动转移到日志文本框
4. 使用 `n` 跳转到下一个匹配项
5. 使用 `N` 跳转到上一个匹配项
6. 按 `Esc` 退出搜索模式

## 技术实现

### 核心函数

- `startLogSearch()`: 初始化搜索模式
- `performLogSearch()`: 执行搜索和高亮
- `highlightLogSearchMatches()`: 高亮匹配项
- `nextLogSearchMatch()`: 下一个匹配项
- `prevLogSearchMatch()`: 上一个匹配项
- `exitLogSearch()`: 退出搜索模式
- `scrollToLogSearchMatch()`: 滚动到匹配项

### 状态变量

- `logSearchActive`: 搜索模式是否激活
- `logSearchQuery`: 当前搜索查询
- `logSearchMatches`: 所有匹配项的位置
- `logSearchCurrentIdx`: 当前匹配项索引
- `logOriginalText`: 原始日志文本
- `logSearchInput`: 搜索输入框组件 