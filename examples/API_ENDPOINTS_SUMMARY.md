# GoHook 日志系统 API 接口总结

## 概述

GoHook 日志系统已经完成调整，移除了 `/api/v1` 前缀，并根据不同模块类型区分了日志接口。

## 更新后的接口列表

### 1. Webhook 日志（用户手动建立规则和脚本的webhook）

| 方法 | 路径 | 功能 |
|------|------|------|
| GET | `/logs/webhooks` | 获取 Webhook 执行日志列表 |
| GET | `/logs/webhooks/stats` | 获取 Webhook 执行统计信息 |

### 2. GitHook 日志（简易版本的githook）

| 方法 | 路径 | 功能 |
|------|------|------|
| GET | `/logs/githook` | 获取 GitHook 执行日志列表 |
| GET | `/logs/githook/stats` | 获取 GitHook 执行统计信息 |

### 3. 用户活动日志

| 方法 | 路径 | 功能 |
|------|------|------|
| GET | `/logs/users` | 获取用户活动记录列表 |
| GET | `/logs/users/stats` | 获取用户活动统计信息 |

### 4. 系统日志

| 方法 | 路径 | 功能 |
|------|------|------|
| GET | `/logs/system` | 获取系统日志列表 |

### 5. 项目活动日志

| 方法 | 路径 | 功能 |
|------|------|------|
| GET | `/logs/projects` | 获取项目活动记录列表 |

### 6. 日志管理

| 方法 | 路径 | 功能 |
|------|------|------|
| DELETE | `/logs/cleanup` | 手动清理旧日志 |

## 主要变更

1. **移除前缀**: 所有接口移除了 `/api/v1` 前缀
2. **模块分离**: 
   - Webhook 和 GitHook 分别使用不同的接口路径
   - 用户活动、系统日志、项目活动各自独立
3. **Hook类型区分**: 
   - 数据库模型中添加了 `hook_type` 字段
   - 支持 "webhook" 和 "githook" 两种类型
4. **统计接口**: 为 Webhook、GitHook 和用户活动添加了专门的统计接口

## 测试示例

```bash
# 获取 Webhook 日志
curl -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  "http://localhost:9000/logs/webhooks?page=1&page_size=20"

# 获取 GitHook 日志
curl -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  "http://localhost:9000/logs/githook?success=true"

# 获取用户活动
curl -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  "http://localhost:9000/logs/users?username=admin"

# 获取系统日志
curl -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  "http://localhost:9000/logs/system?level=ERROR"

# 获取项目活动
curl -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  "http://localhost:9000/logs/projects?project_name=my-project"

# 清理日志
curl -X DELETE -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  "http://localhost:9000/logs/cleanup?days=30"
```

## 运行测试

可以运行以下命令测试数据库功能：

```bash
cd examples
go run test_database_updated.go
```

这将验证：
- 数据库初始化和迁移
- 创建示例日志数据（包括 Webhook 和 GitHook）
- 查询各种类型的日志
- 统计功能

## 注意事项

1. 所有日志接口都需要身份验证
2. 支持分页查询（page, page_size）
3. 支持时间范围过滤（start_time, end_time）
4. 支持各种业务字段过滤
5. 返回格式为 JSON，包含数据和分页信息 