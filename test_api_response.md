# API Response Test

## Pipeline List API Response Format

根据阿里云DevOps API文档，流水线列表API的响应格式如下：

### 请求
```
GET https://{domain}/oapi/v1/flow/organizations/{organizationId}/pipelines?page=1&perPage=30
```

### 响应头
```
x-next-page: 3
x-page: 2  
x-per-page: 10
x-prev-page: 1
x-total: 100
x-total-pages: 10
```

### 响应体
```json
[
    {
        "createAccountId": "22121222",
        "createTime": 1729178040000,
        "pipelineId": 124,
        "pipelineName": "流水线"
    }
]
```

## 修复的问题

1. **分页参数**: 使用正确的 `perPage` 和 `page` 参数
2. **分页检测**: 使用响应头 `x-total-pages` 和 `x-page` 来判断是否还有更多页面
3. **最大每页数量**: 设置为30（API文档限制）
4. **运行历史API**: 尝试多个可能的端点来获取运行历史数据 