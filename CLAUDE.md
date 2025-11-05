# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

GoHook is a lightweight webhook server with a modern web UI, combining:
- **Core webhook functionality** from [adnanh/webhook](https://github.com/adnanh/webhook)
- **Web UI** adapted from [gotify/server](https://github.com/gotify/server)

The project enables users to create HTTP endpoints that execute configured commands, with real-time monitoring through WebSocket connections and a React-based management interface.

## Development Commands

### Backend (Go)

```bash
# Run in development mode
go run .

# Build for local platform
go build -o gohook .

# Run tests
go test -v ./...
make test

# Build for all platforms
make build

# Build specific platform
make build-linux-amd64
make build-linux-arm64
make build-windows-amd64
make build-darwin-amd64
```

### Frontend (React/TypeScript)

```bash
# IMPORTANT: Use yarn, not npm
cd ui

# Install dependencies
yarn

# Start development server (proxies to :9000)
yarn start

# Build production UI
yarn build

# Lint and format
yarn lint
yarn format
yarn testformat
```

### Running the Server

```bash
# Basic run with hooks file
./gohook -hooks hooks.json -verbose

# With HTTPS
./gohook -hooks hooks.json -secure -cert cert.pem -key key.pem

# With hot reload
./gohook -hooks hooks.json -hotreload

# Debug mode
./gohook -hooks hooks.json -verbose -debug -gin-debug

# Custom port
./gohook -hooks hooks.json -port 8080
```

## Architecture

### Backend Structure

**Main entry point**: `app.go`
- Initializes Gin router, loads configurations, sets up middleware
- Registers webhook handler at `/{urlprefix}/*id` (default: `/hooks/*id`)
- Serves embedded React UI from `ui/build` via `ui/serve.go`
- Manages hot-reload using fsnotify for hooks file changes

**Internal packages** (`internal/`):

- **`webhook/`**: Core webhook functionality
  - `hook.go`: Hook struct definition, parameter parsing, signature validation
  - `hooks.go`: Hook collection management, CRUD operations via REST API
  - `request.go`: HTTP request parsing (JSON/XML/form data)
  - Hook execution orchestrated through `HandleHook()` function

- **`router/`**: Gin routing and API setup
  - `router.go`: Initializes router, loads configs, registers middleware
  - `logs.go`: Log query API endpoints (`/api/logs`)
  - `system.go`: System info endpoints

- **`database/`**: Persistence layer using GORM
  - Supports SQLite (default) with automatic CGO/pure-Go driver fallback
  - `models.go`: ExecutionLog model for webhook execution history
  - `service.go`: Log CRUD operations with automatic cleanup
  - Auto-migration on startup

- **`stream/`**: WebSocket manager
  - `stream.go`: Manages client connections, broadcasts webhook events
  - Message types: hook triggered, execution logs, system events
  - Global singleton: `stream.Global`

- **`config/`**: Configuration management
  - Loads `app.yaml` (server config), `user.yaml` (authentication), `version.yaml`
  - Auto-creates default configs if missing

- **`client/`**: User authentication
  - JWT-based auth with configurable expiry
  - Password hashing using bcrypt
  - Default credentials: admin/admin123

- **`middleware/`**: Gin middleware
  - IP detection for proxy environments (X-Forwarded-For, X-Real-IP)
  - JWT validation
  - CORS headers

### Frontend Structure

**Tech stack**: React 19, TypeScript, Material-UI v7, MobX 5

**Source** (`ui/src/`):
- **`app/`**: Main application component
- **`hook/`**: Webhook management views
- **`logs/`**: Execution log viewer
- **`message/`**: Real-time WebSocket message handling
- **`user/`**: Authentication components
- **`system/`**: System info and configuration
- **`CurrentUser.ts`**: MobX store for authentication state
- **`apiAuth.ts`**: Axios instance with JWT interceptor

**Build process**:
- Built React app is embedded into Go binary via `//go:embed` in `ui/serve.go`
- Config injected at runtime by replacing `%CONFIG%` placeholder in `index.html`

### Configuration Files

**`hooks.json`** (or `.yaml`): Webhook definitions
```json
[
  {
    "id": "my-webhook",
    "execute-command": "/path/to/script.sh",
    "command-working-directory": "/path",
    "http-methods": ["POST"],
    "pass-arguments-to-command": [...],
    "trigger-rule": {...}
  }
]
```

**`app.yaml`**: Server configuration
- Port, JWT settings, database config, log retention

**`user.yaml`**: User credentials
- Auto-generated with default admin user if missing

**`version.yaml`**: Version metadata

### Request Flow

1. HTTP request → Gin router → `ginHookHandler()` in `app.go`
2. Extract hook ID from path, match against loaded hooks
3. Parse request body/headers/query based on Content-Type
4. Evaluate trigger rules (signature validation, value matching, etc.)
5. If triggered:
   - Execute command with parsed parameters
   - Log execution to database
   - Broadcast event via WebSocket
   - Return response (synchronous or async based on config)

### WebSocket Communication

- Endpoint: `ws://host:port/api/stream`
- Client connects → added to `stream.Global.clients`
- Server broadcasts: hook triggered events, execution updates
- Heartbeat mechanism tracks connection health

### Database Schema

**ExecutionLog table**:
- Hook ID, execution time, success status, output, error, client IP

Automatic cleanup: Logs older than configured retention period deleted daily.

## Development Notes

- **Go version**: 1.24+ (see `go.mod`)
- **Run backend**: `go run .` (listens on port 9000 by default)
- **Frontend dev**: Proxy configured to `http://localhost:9000` in `ui/package.json`
- **Default login**: admin/admin123 → `/client` endpoint
- **UI theming**: Use dark theme, avoid white backgrounds, prioritize theme colors for consistency

## Testing

- Go tests: `go test -v ./...` or `make test`
- Frontend tests: `cd ui && yarn test`
- Hook testing: Use `test/hookecho.go` as test command

## Build and Deploy

1. Build frontend: `cd ui && yarn build`
2. Build backend: `go build -o gohook .` (embeds UI automatically)
3. Deploy single binary with config files

Cross-platform builds available via Makefile targets.
- 现在gohook经常遇到一个问题，就是如果意外调整了 服务器上文件的权限，githook可能会因为冲突，导致githook同步或者更新代码失败，另外一种情况下，文件在服务器上被修改了，也会因为冲突，导致无法切换版本或者同步代码，这个时候，我们的做法是手动在服务器上放弃修改，使用git reset --hard ，然后手动git pull,这样会导致代码无法及时同步，现在需要添加一个机制，让用户可以选择是否强制同步或者切换