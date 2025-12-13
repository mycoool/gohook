# GoHook - 带Web UI的Webhook服务器

**GoHook** 是一个基于 [webhook](https://github.com/adnanh/webhook) 和 [gotify](https://github.com/gotify/server) 的轻量级可配置工具，使用Go语言编写。它不仅保留了webhook的所有核心功能，还集成了来自gotify项目的现代化Web UI界面，让您可以通过友好的图形界面管理和监控webhook。

## 核心特性

- 🎯 **轻量级HTTP端点**: 在服务器上轻松创建HTTP端点(hooks)来执行配置的命令
- 🌐 **现代化Web UI**: 集成gotify的Web界面，提供直观的管理和监控体验
- 📊 **实时监控**: 通过WebSocket实时查看webhook执行状态和日志
- 🔧 **灵活配置**: 支持JSON和YAML配置文件
- 🔒 **安全规则**: 支持多种触发规则来保护您的端点
- 📡 **数据传递**: 可以将HTTP请求数据(headers、payload、query参数)传递给命令
- 🔄 **热重载**: 支持配置文件热重载，无需重启服务

## 项目背景

本项目基于两个优秀的开源项目：
- **[webhook](https://github.com/adnanh/webhook)**: 提供核心的webhook功能
- **[gotify](https://github.com/gotify/server)**: 提供现代化的Web UI界面

通过结合这两个项目的优势，GoHook为用户提供了既强大又易用的webhook解决方案。

## 快速开始

### 安装

#### 从源码构建
确保您已正确设置Go 1.21或更新版本的环境，然后运行：
```bash
$ go build github.com/mycoool/gohook
```

#### 下载预编译二进制文件
在 [GitHub Releases](https://github.com/mycoool/gohook/releases) 页面下载适合您架构的预编译二进制文件。

### 配置

创建一个名为 `hooks.json` 的配置文件。该文件包含一个hooks数组，定义GoHook将要服务的端点。

简单的hook配置示例：
```json
[
  {
    "id": "redeploy-webhook",
    "execute-command": "/var/scripts/redeploy.sh",
    "command-working-directory": "/var/webhook"
  }
]
```

**YAML格式示例**:
```yaml
- id: redeploy-webhook
  execute-command: "/var/scripts/redeploy.sh"
  command-working-directory: "/var/webhook"
```

### 启动服务

```bash
$ ./gohook -hooks hooks.json -verbose
```

服务将在默认端口9000启动，提供以下功能：

- **Webhook端点**: `http://yourserver:9000/hooks/redeploy-webhook`
- **Web UI界面**: `http://yourserver:9000/` (管理和监控界面)
- **WebSocket**: 实时状态更新和日志推送

## Web UI功能

集成的Web界面提供以下功能：

- 📋 **Hook列表**: 查看所有配置的webhook
- 📊 **执行历史**: 查看webhook执行历史和状态
- 📝 **实时日志**: 通过WebSocket实时查看执行日志
- ⚙️ **配置管理**: 在线查看和管理hook配置
- 📈 **统计信息**: 查看webhook调用统计

## 高级功能

### HTTPS支持
使用 `-secure` 标志启用HTTPS：
```bash
$ ./gohook -hooks hooks.json -secure -cert /path/to/cert.pem -key /path/to/key.pem
```

### 反向代理支持
GoHook可以在反向代理(如Nginx、Apache)后运行，支持TCP端口或Unix域套接字。

### CORS支持
使用 `-header` 标志设置CORS头：
```bash
$ ./gohook -hooks hooks.json -header "Access-Control-Allow-Origin=*"
```

### 模板支持
使用 `-template` 参数将配置文件作为Go模板解析。

## 配置文档

- [Hook定义](docs/Hook-Definition.md) - 详细的hook属性说明
- [Hook规则](docs/Hook-Rules.md) - 触发规则配置
- [Hook示例](docs/Hook-Examples.md) - 复杂配置示例
- [Webhook参数](docs/Webhook-Parameters.md) - 命令行参数说明
- [模板使用](docs/Templates.md) - 模板功能详解

## Docker支持

即将支持

## 社区贡献

即将支持

## 需要帮助？

查看 [现有问题](https://github.com/mycoool/gohook/issues) 或 [创建新问题](https://github.com/mycoool/gohook/issues/new)。


### MIT License

```
MIT License

Copyright (c) 2025 GoHook Contributors

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

### 致谢

本项目基于以下优秀的开源项目：
- [webhook](https://github.com/adnanh/webhook) - MIT License
- [gotify](https://github.com/gotify/server) - MIT License

感谢这些项目的贡献者们的辛勤工作！

---

*GoHook 结合了 webhook 的强大功能和 gotify 的优雅界面，为您提供完整的webhook管理解决方案。*
