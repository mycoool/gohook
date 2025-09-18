# GoHook 数据库日志系统

## 概述

GoHook 现在集成了基于 GORM 的数据库日志系统，支持详细的日志记录和便捷的查看功能。

## 功能特性

- **Hook 执行日志**: 记录每次 webhook 调用的详细信息
- **系统日志**: 记录系统级别的操作和事件
- **用户活动日志**: 跟踪用户的登录、操作等活动
- **项目活动日志**: 记录项目相关的操作（分支切换、标签切换等）
- **自动日志清理**: 定期清理过期日志
- **RESTful API**: 提供完整的日志查询和统计接口

## 数据库配置

### 默认配置

系统默认使用 SQLite 数据库，配置文件位于 `app.yaml`：

```yaml
port: 9000
jwt_secret: "gohook-secret-key-change-in-production"
jwt_expiry_duration: 24
mode: "test"
database:
  type: "sqlite"
  database: "gohook.db"
  log_retention_days: 30
```

### 配置选项

- `type`: 数据库类型（当前支持 sqlite）
- `database`: 数据库文件路径（SQLite）或数据库名称
- `host`: 数据库主机（MySQL/PostgreSQL）
- `port`: 数据库端口
- `username`: 数据库用户名
- `password`: 数据库密码
- `log_retention_days`: 日志保留天数，超过此时间的日志将被自动清理

## 数据模型

### HookLog - Hook执行日志
```go
type HookLog struct {
    ID          uint      `json:"id"`
    CreatedAt   time.Time `json:"created_at"`
    HookID      string    `json:"hook_id"`      // Hook ID
    HookName    string    `json:"hook_name"`    // Hook名称
    HookType    string    `json:"hook_type"`    // Hook类型: "webhook" 或 "githook"
    Method      string    `json:"method"`       // HTTP方法
    RemoteAddr  string    `json:"remote_addr"`  // 客户端IP
    Success     bool      `json:"success"`      // 是否成功
    Duration    int64     `json:"duration"`     // 执行时长（毫秒）
    Output      string    `json:"output"`       // 执行输出
    Error       string    `json:"error"`        // 错误信息
    // ... 更多字段
}
```

**Hook类型说明:**
- `webhook`: 用户手动建立规则和脚本的webhook
- `githook`: 简易版本的githook

### SystemLog - 系统日志
```go
type SystemLog struct {
    ID        uint      `json:"id"`
    CreatedAt time.Time `json:"created_at"`
    Level     string    `json:"level"`     // DEBUG, INFO, WARN, ERROR
    Category  string    `json:"category"`  // AUTH, CONFIG, DATABASE, etc.
    Message   string    `json:"message"`   // 日志消息
    UserID    string    `json:"user_id"`   // 相关用户ID
    IPAddress string    `json:"ip_address"` // IP地址
    // ... 更多字段
}
```

### UserActivity - 用户活动
```go
type UserActivity struct {
    ID          uint      `json:"id"`
    CreatedAt   time.Time `json:"created_at"`
    Username    string    `json:"username"`    // 用户名
    Action      string    `json:"action"`      // 操作类型
    Resource    string    `json:"resource"`    // 操作资源
    Success     bool      `json:"success"`     // 是否成功
    IPAddress   string    `json:"ip_address"`  // IP地址
    // ... 更多字段
}
```

### ProjectActivity - 项目活动
```go
type ProjectActivity struct {
    ID          uint      `json:"id"`
    CreatedAt   time.Time `json:"created_at"`
    ProjectName string    `json:"project_name"` // 项目名称
    Action      string    `json:"action"`       // 操作类型
    OldValue    string    `json:"old_value"`    // 旧值
    NewValue    string    `json:"new_value"`    // 新值
    Username    string    `json:"username"`     // 操作用户
    Success     bool      `json:"success"`      // 是否成功
    // ... 更多字段
}
```

## API 接口

所有日志 API 都需要身份验证。根据不同模块分别提供日志接口。

### Webhook日志（用户手动建立规则和脚本的webhook）

#### 获取Webhook日志列表
```
GET /logs/webhooks
```

查询参数：
- `page`: 页码（默认1）
- `page_size`: 每页数量（默认20，最大100）
- `hook_id`: Hook ID过滤
- `hook_name`: Hook名称过滤（支持模糊匹配）
- `success`: 成功状态过滤（true/false）
- `start_time`: 开始时间（ISO 8601格式）
- `end_time`: 结束时间（ISO 8601格式）

示例：
```bash
curl -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  "http://localhost:9000/logs/webhooks?page=1&page_size=20&success=true"
```

#### 获取Webhook日志统计
```
GET /logs/webhooks/stats
```

查询参数：
- `start_time`: 开始时间
- `end_time`: 结束时间

### GitHook日志（简易版本的githook）

#### 获取GitHook日志列表
```
GET /logs/githook
```

查询参数：
- `page`: 页码（默认1）
- `page_size`: 每页数量（默认20，最大100）
- `hook_id`: Hook ID过滤
- `hook_name`: Hook名称过滤（支持模糊匹配）
- `success`: 成功状态过滤（true/false）
- `start_time`: 开始时间（ISO 8601格式）
- `end_time`: 结束时间（ISO 8601格式）

示例：
```bash
curl -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  "http://localhost:9000/logs/githook?page=1&page_size=20"
```

#### 获取GitHook日志统计
```
GET /logs/githook/stats
```

查询参数：
- `start_time`: 开始时间
- `end_time`: 结束时间

### 用户活动日志

#### 获取用户活动记录
```
GET /logs/users
```

查询参数：
- `page`, `page_size`: 分页参数
- `username`: 用户名过滤
- `action`: 操作类型过滤
- `success`: 成功状态过滤
- `start_time`, `end_time`: 时间范围

示例：
```bash
curl -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  "http://localhost:9000/logs/users?username=admin&action=LOGIN"
```

#### 获取用户活动统计
```
GET /logs/users/stats
```

查询参数：
- `username`: 用户名过滤
- `start_time`: 开始时间
- `end_time`: 结束时间

### 系统日志

#### 获取系统日志列表
```
GET /logs/system
```

查询参数：
- `page`, `page_size`: 分页参数
- `level`: 日志级别过滤（DEBUG, INFO, WARN, ERROR）
- `category`: 日志分类过滤
- `user_id`: 用户ID过滤
- `start_time`, `end_time`: 时间范围

示例：
```bash
curl -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  "http://localhost:9000/logs/system?level=ERROR"
```

### 项目活动日志

#### 获取项目活动记录
```
GET /logs/projects
```

查询参数：
- `page`, `page_size`: 分页参数
- `project_name`: 项目名称过滤
- `action`: 操作类型过滤
- `username`: 用户名过滤
- `success`: 成功状态过滤
- `start_time`, `end_time`: 时间范围

示例：
```bash
curl -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  "http://localhost:9000/logs/projects?project_name=my-project"
```

### 日志管理

#### 手动清理旧日志
```
DELETE /logs/cleanup?days=30
```

查询参数：
- `days`: 保留天数（默认30天）

示例：
```bash
curl -X DELETE -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  "http://localhost:9000/logs/cleanup?days=30"
```

## 自动日志记录

系统会自动记录以下事件：

### Hook执行日志
- 每次 webhook 调用都会自动记录
- 包含请求详情、执行结果、耗时等信息

### 系统日志
- 应用启动/停止
- 配置文件加载/重载
- 数据库连接状态
- 错误和异常

### 用户活动
- 用户登录/登出
- 用户管理操作
- 密码修改

### 项目活动
- 分支切换
- 标签切换
- 项目添加/删除/更新

## 日志级别和分类

### 日志级别
- `DEBUG`: 调试信息
- `INFO`: 一般信息
- `WARN`: 警告信息
- `ERROR`: 错误信息

### 日志分类
- `AUTH`: 身份验证相关
- `CONFIG`: 配置相关
- `DATABASE`: 数据库相关
- `HOOK`: Hook执行相关
- `PROJECT`: 项目操作相关
- `SYSTEM`: 系统级操作
- `API`: API调用相关

## 性能优化

1. **索引优化**: 关键字段已添加数据库索引
2. **分页查询**: 所有列表接口都支持分页
3. **定期清理**: 自动清理过期日志，避免数据库过大
4. **异步记录**: 日志记录不阻塞主要业务流程

## 故障排除

### 数据库连接失败
如果配置的数据库连接失败，系统会自动回退到默认的 SQLite 配置。

### 日志记录失败
如果日志记录失败，系统会在控制台输出错误信息，但不会影响主要功能。

### 数据库文件权限
确保 GoHook 进程对数据库文件所在目录有读写权限。

## 示例用法

### 查看最近的Hook执行记录
```bash
curl -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  "http://localhost:9000/logs/hooks?page=1&page_size=10" | jq '.'
```

### 查看失败的Hook执行
```bash
curl -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  "http://localhost:9000/logs/hooks?success=false" | jq '.'
```

### 查看特定时间范围的系统日志
```bash
curl -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  "http://localhost:9000/logs/system?start_time=2023-12-01T00:00:00Z&end_time=2023-12-02T00:00:00Z" | jq '.'
```

### 获取Hook执行统计
```bash
curl -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  "http://localhost:9000/logs/hooks/stats" | jq '.'
```

## 注意事项

1. 所有API调用都需要有效的JWT令牌
2. 时间参数请使用ISO 8601格式（如：2023-12-01T15:30:00Z）
3. 日志会自动清理，请根据需要调整保留天数
4. 大量日志数据可能影响查询性能，建议使用适当的时间范围和分页参数 