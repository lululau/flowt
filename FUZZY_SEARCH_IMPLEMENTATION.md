# Fuzzy Search 功能实现总结

## 实现概述

根据用户需求，成功实现了以下两个功能：
1. 将流水线搜索改为 Fuzzy Search 方式
2. 为 Pipeline Groups 表格添加 `/` 过滤功能

## 功能详情

### 1. Fuzzy Search 算法

实现了一个简单而高效的 fuzzy search 算法：

```go
func fuzzyMatch(query, text string) bool {
    if query == "" {
        return true
    }
    
    query = strings.ToLower(query)
    text = strings.ToLower(text)
    
    queryIndex := 0
    for _, char := range text {
        if queryIndex < len(query) && rune(query[queryIndex]) == char {
            queryIndex++
        }
    }
    
    return queryIndex == len(query)
}
```

**算法特点**：
- **顺序匹配**：查询字符必须按顺序出现在目标文本中
- **大小写不敏感**：自动转换为小写进行匹配
- **跳跃匹配**：允许字符之间有间隔
- **高效性能**：单次遍历，时间复杂度 O(n)

**匹配示例**：
- `"bp"` 匹配 `"Build Pipeline"` ✓ (首字母匹配)
- `"pipe"` 匹配 `"Build Pipeline"` ✓ (子串匹配)
- `"devenv"` 匹配 `"Development Environment"` ✓ (跨词匹配)
- `"abc"` 匹配 `"axbxcx"` ✓ (间隔匹配)

### 2. 流水线搜索升级

**修改前**：使用简单的字符串包含匹配
```go
if strings.Contains(strings.ToLower(p.Name), sqLower) || 
   strings.Contains(strings.ToLower(p.PipelineID), sqLower) {
    // 匹配
}
```

**修改后**：使用 fuzzy search
```go
if fuzzyMatch(currentSearchQuery, p.Name) || 
   fuzzyMatch(currentSearchQuery, p.PipelineID) {
    // 匹配
}
```

**用户体验提升**：
- 可以使用首字母快速搜索：`"bp"` 找到 `"Build Pipeline"`
- 支持不连续字符匹配：`"bldpipe"` 找到 `"Build Pipeline"`
- 更灵活的搜索方式，减少输入量

### 3. Pipeline Groups 搜索功能

为 Pipeline Groups 页面添加了完整的搜索功能：

#### 新增组件
- **搜索输入框**：与流水线页面风格一致的搜索框
- **搜索状态管理**：`currentGroupSearchQuery` 全局变量
- **搜索输入框引用**：`groupSearchInputGlobal` 用于焦点管理

#### UI 布局更新
```go
groupListFlexView := tview.NewFlex().SetDirection(tview.FlexRow).
    AddItem(groupSearchInput, 1, 1, false).    // 新增搜索框
    AddItem(groupTable, 0, 1, true).
    AddItem(groupHelpInfo, 1, 1, false)
```

#### 搜索过滤逻辑
```go
// Filter groups by search query (fuzzy search)
filteredGroups := make([]api.PipelineGroup, 0)
if currentGroupSearchQuery != "" {
    for _, g := range allPipelineGroups {
        if fuzzyMatch(currentGroupSearchQuery, g.Name) || 
           fuzzyMatch(currentGroupSearchQuery, g.GroupID) {
            filteredGroups = append(filteredGroups, g)
        }
    }
} else {
    filteredGroups = append(filteredGroups, allPipelineGroups...)
}
```

### 4. 键盘快捷键支持

#### 流水线页面
- **`/`**：聚焦到搜索框
- **`Escape`**：清空搜索并返回表格

#### Groups 页面
- **`/`**：聚焦到 groups 搜索框
- **`Escape`**：清空搜索并返回表格
- **`q`**：返回流水线页面（同时清空搜索状态）

#### 全局快捷键
- **`/`**：根据当前页面自动聚焦到对应的搜索框
  - 在 pipelines 页面：聚焦流水线搜索框
  - 在 groups 页面：聚焦 groups 搜索框

### 5. 状态管理

#### 搜索状态清理
确保在页面切换时正确清理搜索状态：

```go
case 'q':
    // Back to pipelines view
    currentViewMode = "all_pipelines"
    selectedGroupID = ""
    selectedGroupName = ""
    currentGroupSearchQuery = ""        // 清空 groups 搜索
    groupSearchInput.SetText("")        // 清空输入框
    updatePipelineTable(pipelineTable, app, searchInput, apiClient, orgId)
    mainPages.SwitchToPage("pipelines")
    app.SetFocus(pipelineTable)
    return nil
```

#### 输入框焦点管理
在全局键盘事件处理中正确处理搜索框的 'q' 键输入：

```go
case 'q': // Lowercase q
    // If searchInput or groupSearchInput is focused, allow typing 'q'
    if focused == searchInput || focused == groupSearchInputGlobal {
        return event
    }
    // Otherwise consume the event to prevent quit
    return nil
```

## 技术实现细节

### 1. 事件处理器链
- **组件级别**：各个表格和输入框的专用事件处理
- **页面级别**：mainPages 的全局快捷键处理
- **应用级别**：app 的最终事件处理（退出等）

### 2. 焦点管理
- 搜索框完成输入后自动返回表格
- 页面切换时正确设置焦点
- 模态框关闭后恢复到合适的组件

### 3. 样式一致性
- Groups 搜索框与流水线搜索框使用相同的样式
- 透明背景，白色文字，灰色占位符
- 一致的标签和提示文本

## 用户使用指南

### 流水线搜索（Fuzzy Search）
1. 在流水线列表页面按 `/` 键
2. 输入搜索关键词，支持：
   - 首字母：`"bp"` 找 `"Build Pipeline"`
   - 部分字符：`"pipe"` 找 `"Build Pipeline"`
   - 跨词匹配：`"devenv"` 找 `"Development Environment"`
3. 按 `Enter` 或方向键返回表格
4. 按 `Escape` 清空搜索

### Groups 搜索
1. 按 `Ctrl+G` 进入 Groups 页面
2. 按 `/` 键聚焦搜索框
3. 输入 group 名称或 ID 进行 fuzzy search
4. 按 `Enter` 返回表格选择
5. 按 `q` 返回流水线页面

## 测试验证

创建了完整的测试用例验证 fuzzy search 算法：
- 空查询匹配所有内容 ✓
- 顺序字符匹配 ✓
- 大小写不敏感 ✓
- 间隔字符匹配 ✓
- 错误顺序不匹配 ✓
- 缺失字符不匹配 ✓
- 实际使用场景测试 ✓

## 兼容性

- 完全向后兼容现有功能
- 不影响其他页面的操作
- 保持原有的键盘快捷键
- 维持一致的用户体验

## 总结

成功实现了用户要求的两个功能：

✅ **Fuzzy Search**：流水线搜索升级为更智能的模糊匹配  
✅ **Groups 搜索**：Pipeline Groups 表格支持 `/` 键过滤功能  
✅ **一致体验**：两个搜索功能使用相同的 fuzzy search 算法  
✅ **完整测试**：算法经过全面测试验证  
✅ **用户友好**：直观的快捷键和清晰的状态管理  

这些改进大大提升了用户在大量流水线和分组中快速定位目标的效率。 