<!-- generated-by: gsd-doc-writer -->

# Deployment

RSS Reader is designed for single-user, self-hosted deployment. The primary deployment method is Docker Compose, which runs the Go backend and Nuxt frontend in separate containers with persistent storage.

## Deployment Targets

| Target | Config File | Notes |
|--------|-------------|-------|
| Docker Compose (PostgreSQL + pgvector) | `docker-compose.yml` | **默认/推荐方式**，PostgreSQL 内置 pgvector 扩展支持向量检索，包含前后端 + 数据库三个服务。 |
| Docker Compose (SQLite) | `docker-compose.sqlite.yml` | 仅在 `sqlite` 分支可用，主分支不再维护。 |

No PaaS-specific config files (Vercel, Netlify, Fly.io, etc.) are present. The application is intended to run on a single host via Docker Compose.

> **注意：SQLite 版本已归档到独立的 `sqlite` 分支，如需使用请切换到该分支后部署。主分支仅支持 PostgreSQL 数据库。**

## Build Pipeline

No CI/CD pipeline is configured — there are no `.github/workflows/` files in the repository. Build and deploy are manual steps.

### Container Build Process

Both Dockerfiles use multi-stage builds:

**Backend** (`backend-go/Dockerfile`):
1. Stage `build`: `golang:1.25-alpine` — downloads Go modules, compiles `cmd/server` into a static binary (`CGO_ENABLED=0`).
2. Final: `alpine:3.22` — copies the binary and `configs/` directory, runs as non-root user `appuser` (UID 10001).

**Frontend** (`front/Dockerfile`):
1. Stage `build`: `node:22-alpine` — installs pnpm via corepack, runs `pnpm install --frozen-lockfile`, then `pnpm build`.
2. Final: `node:22-alpine` — copies `.output/` from the build stage, runs `node .output/server/index.mjs`.

### Docker Compose (PostgreSQL + pgvector) — Quick Deploy (Default)

```bash
cp .env.example .env
docker compose up --build -d
```

This starts three services:

- **postgres**: PostgreSQL 17 with pgvector extension on port 5432, data persisted via `pgdata` volume.
- **backend**: Go API server on port 5000, connects to postgres service internally.
- **front**: Nuxt SSR server on internal port 3000, mapped to a host port via `${FRONT_PORT:-3000}` (the `.env.example` sets `FRONT_PORT=3000`). Proxies API calls to the backend container internally via `http://backend:5000/api`.

After startup (with default `.env.example`):
- Frontend: `http://localhost:3000`
- Backend API: `http://localhost:5000/api`

### Docker Compose (SQLite)
> 仅在 `sqlite` 分支可用
```bash
cp .env.example .env
docker compose -f docker-compose.sqlite.yml up --build -d
```

This starts two services:

- **backend**: Go API server on port 5000, SQLite data at `/app/data/` inside the container, mounted from `./data/` on the host.
- **front**: Nuxt SSR server on internal port 3000, mapped to a host port via `${FRONT_PORT:-3000}` (the `.env.example` sets `FRONT_PORT=3000`). Proxies API calls to the backend container internally via `http://backend:5000/api`.

After startup (with default `.env.example`):
- Frontend: `http://localhost:3000`
- Backend API: `http://localhost:5000/api`

```bash
cp .env.example .env
# Edit .env to set POSTGRES_PASSWORD for production
docker compose -f docker-compose.yml up --build -d
```

This adds a third service:

- **postgres**: `pgvector/pgvector:pg18-trixie` image with a health check (`pg_isready`). Data persisted in `./data/` on the host. The init script at `docker/postgres/init/01-enable-pgvector.sql` runs `CREATE EXTENSION IF NOT EXISTS vector` on first start.

To connect the backend to PostgreSQL, you also need to set the backend environment variables (either in a custom compose overlay or by editing the compose file):

```yaml
environment:
  DATABASE_DRIVER: postgres
  DATABASE_DSN: "host=postgres user=postgres password=postgres dbname=rss_reader sslmode=disable"
```

## Environment Setup

See [configuration.md](configuration.md) for the full list of environment variables with defaults and descriptions.

### Minimum Required for Docker Deploy

The `.env.example` file contains three variables:

```bash
FRONT_PORT=3000
BACKEND_PORT=5000
SQLITE_DB_FILE=rss_reader.db
```

All values have defaults — the application starts with zero configuration. The only scenario that causes a startup failure is an invalid or unreachable database DSN.

### Production Considerations

For a production deployment, review these settings:

| Variable | Why it matters |
|---|---|
| `SERVER_MODE` | Set to `"release"` in Docker Compose to suppress Gin debug output. Defaults to `"debug"` outside Docker. |
| `CORS_ORIGINS` | Must include the origin where users access the frontend (e.g., `http://your-host:3000`). |
| `POSTGRES_PASSWORD` | Change from the default `"postgres"` if using the PostgreSQL compose file. |
| `NUXT_PUBLIC_API_ORIGIN` | Must match the externally reachable backend URL. |
| `NUXT_PUBLIC_API_BASE` | Must match the externally reachable API URL. |

AI-related settings (LLM credentials, Firecrawl, digest export) are configured through the web UI and stored in the database — they are not set via environment variables. See [configuration.md](configuration.md#database-stored-settings-ai-features) for details.

### Proxy Settings (China / Restricted Networks)

Both Dockerfiles accept build-arg proxies for dependency downloads:

```bash
# In .env or shell environment
GOPROXY=https://goproxy.cn,direct
GOSUMDB=sum.golang.google.cn
NPM_CONFIG_REGISTRY=https://registry.npmmirror.com
HTTP_PROXY=http://proxy:port
HTTPS_PROXY=http://proxy:port
```

These are passed through the `build.args` section in both Docker Compose files.

## Data Persistence

### SQLite

The SQLite database file is stored inside the container at `/app/data/` and mounted from `./data/` on the host. The filename defaults to `rss_reader.db` (configurable via `SQLITE_DB_FILE`).

**Backup**: Stop the containers and copy the database file:

```bash
cp ./data/rss_reader.db ./data/rss_reader.db.backup
```

### PostgreSQL

Postgres data is mounted from `./data/` on the host (the `docker-compose.yml` maps `./data/` to `/var/lib/postgresql`).

**Backup**:

```bash
docker exec zanebono-rssreader-pgvector pg_dump -U postgres rss_reader > backup.sql
```

## Rollback Procedure

Since there is no CI/CD pipeline, rollback is manual:

1. Stop the running containers:
   ```bash
   docker compose -f docker-compose.sqlite.yml down
   ```
2. Check out a previous known-good commit:
   ```bash
   git checkout <previous-commit-hash>
   ```
3. Rebuild and restart:
   ```bash
   docker compose -f docker-compose.sqlite.yml up --build -d
   ```

If you tag releases, you can also `git checkout <tag>` instead of a commit hash.

**Database rollback**: SQLite does not support down migrations. Always back up the database file before upgrading. If a new version includes schema changes that break the old backend, restore the backed-up `rss_reader.db` file.

## Monitoring

The backend includes built-in OpenTelemetry tracing with a custom SQLite span exporter. Traces are written to the `otel_spans` table in the same database as application data.

Key tracing details:
- **Library**: `go.opentelemetry.io/otel` with `otelgin` HTTP middleware applied globally to all routes
- **Exporter**: Custom `SQLiteSpanExporter` that writes spans to the `otel_spans` table
- **HTTP middleware**: `otelgin.Middleware("rss-reader-backend")` captures request-level spans for all HTTP handlers
- **Traced scheduler operations**: auto_refresh, firecrawl, content_completion, auto_summary, preference_update
- **Traced domain operations**: AI summary queue batch processing, AI router chat
- **Retention**: 7 days (configurable via `tracing.DefaultConfig()`)
- **Buffer**: 100 spans, flushed every 5 seconds
- **Query API**: The backend exposes trace query endpoints through `internal/platform/tracing/handler.go` — recent traces, trace by ID, timeline view, stats, search by operation/status/duration, and OTLP JSON export

No external monitoring services (Sentry, Datadog, New Relic) are configured. The built-in tracing provides basic observability for feed refresh cycles, AI operations, and HTTP request latency.

To view traces, use the application's built-in tracing UI <!-- VERIFY: tracing UI location in the frontend --> or query the `otel_spans` table directly.
