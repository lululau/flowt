# Flowt - 阿里云云效流水线 TUI 工具

Flowt 是一个基于 Go 语言开发的命令行 TUI（Terminal User Interface）工具，用于管理阿里云云效（DevOps）流水线。它提供了直观的终端界面，让您可以轻松查看、运行和管理流水线。

## 功能特性

- 📋 **流水线列表管理**：以表格形式展示流水线列表，支持 Fuzzy Search 过滤和状态过滤
- 🗂️ **分组视图**：支持按分组查看流水线，可在分组视图和全部视图之间切换
- ▶️ **流水线运行**：一键运行流水线，自动显示实时日志流
- ⏹️ **流水线控制**：停止正在运行的流水线
- 📊 **详细信息**：查看流水线详情，包括阶段、任务等信息
- 📈 **运行历史**：查看流水线运行历史和每次运行的日志
- 🎨 **透明界面**：所有界面背景透明，适配各种终端主题
- ⌨️ **Vim 风格快捷键**：支持 Vim 风格的键盘操作

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

### 配置

创建配置文件 `flowt.yml`：

```yaml
# 阿里云认证配置
aliyun:
  # 方式1：使用 Personal Access Token（推荐）
  personal_access_token: "your_personal_access_token"
  endpoint: "openapi-rdc.aliyuncs.com"
  
  # 方式2：使用 AccessKey（可选）
  # access_key_id: "your_access_key_id"
  # access_key_secret: "your_access_key_secret"
  # region_id: "cn-hangzhou"

# 组织 ID
organization_id: "your_organization_id"
```

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

### 快捷键

- `Tab` / `Shift+Tab` - 在不同面板间切换
- `Enter` - 选择/进入
- `Esc` - 返回上级
- `/` - 搜索过滤
- `r` - 运行流水线
- `s` - 停止流水线
- `d` - 查看详情
- `h` - 查看历史
- `g` - 切换分组视图
- `q` - 退出

## 技术架构

- **UI 框架**：[tview](https://github.com/rivo/tview) - 强大的 TUI 库
- **API 客户端**：阿里云 Go SDK + 自定义 HTTP 客户端
- **配置管理**：YAML 配置文件
- **日志系统**：结构化日志记录

## API 支持

基于阿里云云效 API 实现，支持：

- 流水线管理（列表、详情、运行、停止）
- 流水线分组管理
- 运行历史查询
- 实时日志流
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
├── flowt.yml                    # 配置文件
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

## 贡献

欢迎提交 Issue 和 Pull Request！

## 许可证

MIT License

## 相关文档

- [代理支持说明](PROXY_SUPPORT.md)
- [模糊搜索实现](FUZZY_SEARCH_IMPLEMENTATION.md)
- [流水线运行修复](RUN_PIPELINE_FIX.md)
- [阿里云云效 API 文档](https://help.aliyun.com/zh/yunxiao/developer-reference/)

---

## 原始需求文档

以下是项目的原始需求文档，保留作为参考：

### 功能需求

1. 可以查看流水线列表，以表格形式展示。列表支持 Fuzzy Search 过滤，支持按照流水线状态过滤；接口通常是分页的，流水线列表需要一次性展示全部流水线，不要分页展示。
2. 支持按照分组查看流水线列表（先显示分组列表，在某一个分组上回车进入该分组的流水线列表），也支持不按分组查看（全部）流水线列表，支持通过快捷键在这两种视图之间切换
3. 可以运行流水线，运行流水线之后自动显示当前运行的流水线的日志，并自动刷新日志流
4. 可以停止正在运行的流水线
5. 可以查看流水线的详情，包括流水线的阶段、任务等信息
6. 可以查看流水线的运行历史，包括每次运行的状态、开始时间、结束时间等信息
7. 可以查看每个流水线历史记录的日志。对于当前正在运行的历史记录，支持自动刷新日志流。
8. 所有的界面，不要设置设置背景色，我希望背景色是透明的
9. 所有的界面，要填满整个终端窗口的宽度，不要限制表格等的宽度和高度

### 技术参考

1. TUI 的界面布局和操作方式参考 [k9s](https://github.com/derailed/k9s)，使用 [tview](https://github.com/rivo/tview) 库进行 TUI 界面创建，支持 vim 风格的快捷键

2. 基于阿里云云效 API 实现各项功能

