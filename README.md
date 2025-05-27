# Flowt - 阿里云云效流水线 TUI 工具

Flowt 是一个基于 Go 语言开发的命令行 TUI（Terminal User Interface）工具，用于管理阿里云云效（DevOps）流水线。它提供了直观的终端界面，让您可以轻松查看、运行和管理流水线。

## 功能特性

- 📋 **流水线列表管理**：以表格形式展示流水线列表，支持模糊搜索和状态筛选
- 🔖 **书签功能**：收藏重要流水线，支持书签筛选和优先排序
- 🗂️ **分组视图**：支持按分组查看流水线，可在分组视图和全部视图之间切换
- ▶️ **流水线运行**：一键运行流水线，支持分支选择，自动显示实时日志流
- 📈 **运行历史**：查看流水线运行历史，支持分页浏览和直接查看日志
- 📊 **智能日志显示**：实时日志流，支持自动刷新、手动刷新、编辑器查看和分页器查看
- 🎨 **透明界面**：所有界面背景透明，适配各种终端主题
- ⌨️ **Vim 风格快捷键**：支持 j/k 导航等 Vim 风格的键盘操作

## 网络代理支持

Flowt 支持通过环境变量配置 HTTP/HTTPS 代理，适用于企业网络环境：

```bash
# 设置代理
export http_proxy=http://proxy.company.com:8080
export https_proxy=http://proxy.company.com:8080

# 带认证的代理
export http_proxy=http://username:password@proxy.company.com:8080
export https_proxy=http://username:password@proxy.company.com:8080

# 运行程序
./flowt
```

支持的环境变量：
- `http_proxy` / `HTTP_PROXY` - HTTP 代理服务器
- `https_proxy` / `HTTPS_PROXY` - HTTPS 代理服务器  
- `no_proxy` / `NO_PROXY` - 不使用代理的地址列表

详细的代理配置说明请参考 [PROXY_SUPPORT.md](PROXY_SUPPORT.md)。

## 安装

### 从源码编译

```bash
git clone https://github.com/your-username/flowt.git
cd flowt
go build -o flowt ./cmd/aliyun-pipelines-tui
```

## 配置

### 配置文件位置

创建配置文件 `~/.config/flowt.yml`：

```yaml
# 企业 ID（组织 ID）- 必填
organization_id: "your_organization_id"

# 推荐：使用个人访问令牌认证
personal_access_token: "pt-XXXXXXXXXXXXXXXXXXXXXXXX_XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX"
endpoint: "openapi-rdc.aliyuncs.com"  # 可选，默认值

# 或者：使用AccessKey认证（备用方式）
# access_key_id: "your_access_key_id"
# access_key_secret: "your_access_key_secret"
# region_id: "cn-hangzhou"  # 可选，默认值

# 编辑器和分页器配置（可选）
editor: "nvim"  # 或 "code --wait", "vim" 等
pager: "less -R"  # 或 "lnav", "bat" 等

# 书签配置（可选）
bookmarks:
  - "demo_staging"
  - "demo_prod"
```

### 认证方式

#### 推荐：个人访问令牌认证
- 更安全，权限控制更精细
- 无需管理区域配置
- 获取方式：阿里云云效 → 个人设置 → 个人访问令牌

#### 备用：AccessKey认证
- 传统认证方式
- 需要配置区域（默认 cn-hangzhou）
- 获取方式：阿里云控制台 → AccessKey管理

## 使用方法

```bash
# 启动程序
./flowt

# 启用调试模式
FLOWT_DEBUG=1 ./flowt

# 使用代理
export http_proxy=http://proxy.company.com:8080
./flowt
```

## 快捷键说明

### 主界面（流水线列表）
- `j/k` - 上下移动选择
- `Enter` - 查看运行历史
- `r` - 运行流水线
- `a` - 切换状态筛选（全部 ↔ 运行中+等待中）
- `b` - 切换书签筛选（全部 ↔ 仅书签）
- `B` - 添加/移除书签
- `Ctrl+G` - 切换到分组视图
- `/` - 聚焦搜索框
- `q` - 返回上级/退出
- `Q` - 直接退出程序

### 分组视图
- `j/k` - 上下移动选择
- `Enter` - 进入分组查看流水线
- `/` - 聚焦搜索框
- `q` - 返回流水线列表
- `Q` - 直接退出程序

### 运行历史
- `j/k` - 上下移动选择
- `Enter` - 查看日志
- `r` - 运行流水线
- `[/]` - 上一页/下一页
- `0` - 跳转到第一页
- `q` - 返回流水线列表
- `Q` - 直接退出程序

### 日志查看
- `r` - 手动刷新日志
- `e` - 在编辑器中查看日志
- `v` - 在分页器中查看日志
- `q` - 返回上级界面
- `Q` - 直接退出程序

## 核心功能

### 书签管理
- 使用 `B` 键快速添加/移除流水线书签
- 使用 `b` 键在全部流水线和书签流水线之间切换
- 书签流水线在列表中优先显示（★ 标记）
- 书签自动保存到配置文件

### 状态筛选
- 使用 `a` 键在全部流水线和运行中流水线之间切换
- 支持 RUNNING 和 WAITING 状态的快速筛选
- 与搜索和书签功能完全兼容

### 智能日志显示
- 新创建的运行：自动刷新直到完成
- 历史运行（运行中）：自动刷新直到状态改变
- 历史运行（已完成）：仅显示，不自动刷新
- 状态栏显示运行状态和刷新状态

### 编辑器和分页器支持
- 支持在外部编辑器中查看和编辑日志
- 支持在分页器中浏览长日志
- 配置优先级：配置文件 → 环境变量 → 默认值

## 技术架构

- **UI 框架**：[tview](https://github.com/rivo/tview) - 强大的 TUI 库
- **API 客户端**：阿里云 Go SDK + 自定义 HTTP 客户端
- **配置管理**：YAML 配置文件
- **认证支持**：个人访问令牌（推荐）+ AccessKey（备用）

## API 支持

基于阿里云云效 API 实现，支持：

- 流水线管理（列表、详情、运行、停止）
- 流水线分组管理
- 运行历史查询
- 实时日志流（包括部署日志）
- 任务详情查看

## 开发

### 项目结构

```
flowt/
├── cmd/aliyun-pipelines-tui/    # 主程序入口
├── internal/
│   ├── api/                     # API 客户端
│   └── ui/                      # TUI 界面组件
├── logs/                        # 日志文件
├── flowt.yml.example            # 配置文件示例
└── docs/                        # 文档
```

### 构建

```bash
# 开发构建
go build -o flowt ./cmd/aliyun-pipelines-tui

# 生产构建
go build -ldflags="-s -w" -o flowt ./cmd/aliyun-pipelines-tui
```

### 调试

```bash
# 启用调试日志
export FLOWT_DEBUG=1
./flowt

# 查看 API 调试日志
tail -f logs/api_debug.log
```

## 故障排除

### 常见问题

1. **配置文件未找到**
   - 确保在 `~/.config/flowt.yml` 创建配置文件
   - 参考 `flowt.yml.example` 示例

2. **认证失败**
   - 检查个人访问令牌是否有效
   - 确认组织ID是否正确
   - 验证网络连接和代理设置

3. **界面显示异常**
   - 确保终端支持 Unicode 字符
   - 调整终端窗口大小
   - 检查终端颜色支持

## 贡献

欢迎提交 Issue 和 Pull Request！

## 许可证

MIT License

## 相关文档

- [书签功能说明](BOOKMARK_FEATURE.md)
- [日志状态栏集成](LOG_STATUS_BAR_INTEGRATION.md)
- [代理支持说明](PROXY_SUPPORT.md)
- [模糊搜索实现](FUZZY_SEARCH_IMPLEMENTATION.md)
- [实现总结](IMPLEMENTATION_SUMMARY.md)
- [阿里云云效 API 文档](https://help.aliyun.com/zh/yunxiao/developer-reference/)
