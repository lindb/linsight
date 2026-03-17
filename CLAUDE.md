# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

LinSight is an open-source observability platform (similar to Grafana/DataDog) providing unified metrics, logs, traces, and dashboards. The backend is Go; the frontend is React/TypeScript in `/web`.

## Common Commands

```bash
# Run the server
make run
# or: go run github.com/lindb/linsight/cmd server run

# Run tests (includes lint)
make test

# Run tests without lint
make test-without-lint

# Lint
make lint

# Generate mocks
make gomock

# Add license headers
make header

# Build frontend
make build-frontend

# Update dependencies
make deps
```

**Run a single test:**
```bash
go test ./path/to/package/... -run TestName -race
```

**Database migration:**
```bash
go run github.com/lindb/linsight/cmd migrator all
```
Default credentials after migration: `admin` / `admin`

## Architecture

The project follows a layered architecture:

```
cmd/         → Entry points: server startup, migration runner
config/      → Configuration structs (TOML + env vars with LINSIGHT_* prefix)
http/
  api/       → HTTP handlers (thin layer, delegates to services)
  deps/      → Dependency injection containers
  middleware/ → Auth/authz middleware (InitContext → Authenticate → Authorize)
  router.go  → All route registrations (base path: /api/v1)
service/     → Business logic layer
model/       → GORM data models
pkg/db/      → Database abstraction over GORM (SQLite/MySQL/PostgreSQL)
plugin/      → Datasource plugins (e.g., LinDB)
accesscontrol/ → RBAC + ABAC via Casbin
provisioning/  → Dashboard provisioning from files
```

> **⚠️ Must sync `openapi.yaml`**: Whenever you add, modify, or delete any API endpoint (routes, request parameters, response schemas), you **MUST** update `openapi.yaml` accordingly. The spec must always exactly match the actual implementation.

### Key Patterns

- **Dependency injection**: `http/deps/api.go` holds all service dependencies injected into handlers
- **BaseModel** (`model/base.go`): All models embed this — includes timestamps, `created_by`, `updated_by`
- **Access control**: Two layers — RBAC (`rbac_api.conf`) and ABAC (`abac_resource.conf`). Roles: Lin (super admin) > Admin > Editor > Viewer > Anonymous
- **Authentication**: Cookie-based sessions with token table in DB; password uses salted hashing
- **Database**: Defaults to SQLite (`linsight.db`). Configured via `linsight.toml` or env vars

### Adding or Modifying an API Endpoint

1. Add model in `model/`
2. Add service logic in `service/`
3. Add HTTP handler in `http/api/`
4. Register dependency in `http/deps/api.go`
5. Register route in `http/router.go`
6. Add access control policy if needed in `accesscontrol/`
7. **Always sync `openapi.yaml`** — whenever you add, modify, or remove a route, request body, response schema, or parameter, you **MUST** update `openapi.yaml`. This applies to both **new endpoints and updates to existing ones**. The response schema must exactly match what the handler actually returns.

## Configuration

Default config file: `linsight.toml` in project root. Supports environment variable overrides with `LINSIGHT_` prefix.

Database config example:
```toml
[database]
type = "sqlite"    # or "mysql", "postgres"
dsn = "linsight.db"
debug = true
```
