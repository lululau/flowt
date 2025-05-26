# UI 改进总结

## 修改内容

### 1. 流水线列表页简化
- **修改前**: 显示5列 - Pipeline Name, ID, Status, Creator, Last Run Time
- **修改后**: 只显示2列 - ID (左), Pipeline Name (右)
- **目的**: 简化界面，突出重要信息

### 2. 搜索功能优化
- **快捷键修改**: 从 `Ctrl+F` 改为 `/`
- **背景色优化**: 搜索框背景色改为透明 (`tcell.ColorDefault`)
- **移除状态过滤**: 完全移除 `Ctrl+S` 状态过滤功能
- **相关变量清理**: 移除 `currentStatusFilter`, `statusesToCycle`, `currentStatusIndex` 等变量

### 3. Run History 表格列宽优化
- **列宽分布调整**: 使用 `SetExpansion()` 方法优化列宽比例
  - `#` 列: 最小宽度 (expansion=1), 居中对齐
  - `Status` 列: 小宽度 (expansion=2), 居中对齐
  - `Trigger` 列: 小宽度 (expansion=2), 左对齐, 超过10字符截断
  - `Start Time` 列: 较大宽度 (expansion=3), 左对齐
  - `Finish Time` 列: 较大宽度 (expansion=3), 左对齐
  - `Duration` 列: 最小宽度 (expansion=1), 右对齐

### 4. 快捷键行为统一
- **大写Q**: 退出程序 (`app.Stop()`)
- **小写q**: 退出当前界面，返回上一级
  - 在 groups 页面: 返回 pipelines 页面
  - 在 run_history 页面: 返回 pipelines 页面
  - 在 logs 页面: 返回 run_history 或 pipelines 页面
- **Esc键**: 保持原有行为，与小写q功能相同

## 技术实现细节

### 搜索功能
```go
// 修改前
case tcell.KeyCtrlF:
    if currentPage == "pipelines" {
        app.SetFocus(searchInput)
        return nil
    }

// 修改后
case '/':
    if currentPage == "pipelines" {
        app.SetFocus(searchInput)
        return nil
    }
```

### 列宽优化
```go
// 使用 SetExpansion() 方法控制列宽比例
runNumCell := tview.NewTableCell(fmt.Sprintf("#%d", totalRuns-globalRunIndex)).
    SetTextColor(tcell.ColorLightBlue).
    SetAlign(tview.AlignCenter).
    SetBackgroundColor(tcell.ColorDefault).
    SetExpansion(1) // 最小宽度
```

### 快捷键处理
```go
// 统一的退出逻辑
case 'Q':
    app.Stop()  // 退出程序
    return nil
case 'q':
    // 退出当前界面的逻辑
    if currentPage == "groups" {
        // 返回pipelines页面
    } else if currentPage == "run_history" {
        // 返回pipelines页面
    }
    return nil
```

## 用户体验改进

1. **界面更简洁**: 流水线列表只显示核心信息
2. **搜索更直观**: 使用常见的 `/` 键进行搜索
3. **表格更美观**: Run History 表格列宽分布更合理
4. **操作更一致**: Q/q 键的行为符合常见软件习惯

## 兼容性

- 所有修改都向后兼容
- 不影响现有的API调用
- 保持原有的分页功能
- 保持原有的键盘导航功能

## 最新修复 (v1.2.0)

### 问题修复
1. **搜索框背景色**: 添加了 `SetLabelColor()` 和 `SetFieldTextColor()` 确保搜索框完全透明
2. **分页大小**: Run History 表格分页大小从 10 改为 30
3. **小写q键行为**: 修复了小写q会退出程序的问题
   - 在各个表格的事件处理器中添加了'q'键处理
   - 确保'q'键在表格级别被处理，不会传递到全局处理器
   - 在搜索框中可以正常输入'q'字符

### 技术实现
```go
// 搜索框透明背景
searchInput.SetBackgroundColor(tcell.ColorDefault)
searchInput.SetFieldBackgroundColor(tcell.ColorDefault)
searchInput.SetLabelColor(tcell.ColorWhite)
searchInput.SetFieldTextColor(tcell.ColorWhite)

// 分页大小调整
runHistoryPerPage = 30

// 表格级别的'q'键处理
case 'q':
    // Back to pipelines view
    isRunHistoryActive = false
    mainPages.SwitchToPage("pipelines")
    app.SetFocus(pipelineTable)
    return nil
```

## 测试建议

1. 测试流水线列表的显示和搜索功能
2. 测试 Run History 表格的分页和列宽显示（每页30条记录）
3. 测试所有快捷键的行为是否符合预期：
   - 小写'q'应该返回上一级界面，不退出程序
   - 大写'Q'应该退出程序
   - 在搜索框中可以正常输入'q'字符
4. 测试在不同终端尺寸下的显示效果
5. 测试搜索框的背景色是否完全透明 