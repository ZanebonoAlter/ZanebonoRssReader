<!-- generated-by: gsd-doc-writer -->

# Configuration

The RSS Reader uses a layered configuration system: a YAML config file for the backend, environment variables that override file values, and Nuxt runtime config for the frontend. AI-related settings (LLM, Firecrawl, digest) are stored in the database and configured via the web UI.

## Environment Variables

### Backend (Go)

These environment variables override values from `backend-go/configs/config.yaml`. If unset, the config file defaults apply.

| Variable | Required | Default | Description |
|---|---|---|---|
| `SERVER_PORT` | No | `"5000"` | HTTP port the backend listens on |
| `SERVER_MODE` | No | `"debug"` | Gin mode: `"debug"`, `"release"`, or `"test"` |
| `DATABASE_DRIVER` | No | `"sqlite"` | Database driver: `"sqlite"` or `"postgres"` |
| `DATABASE_DSN` | No | `"rss_reader.db"` | Data source name. For SQLite: file path. For Postgres: connection string |
| `CORS_ORIGINS` | No | `"http://localhost:3001,http://localhost:3000"` | Comma-separated list of allowed CORS origins |
| `CRAWL_SERVICE_URL` | No | `"http://localhost:11235"` | URL for the crawl/content-completion service |
| `REDIS_URL` | No | *(empty)* | Redis URL for the topic analysis job queue. When set, the queue uses Redis as a persistent backend; otherwise falls back to in-memory |

### Topic Analysis Tuning

These environment variables control the AI topic analysis module. They are read at runtime in `internal/domain/topicanalysis/ai_analysis.go` via `parseEnvInt` / `parseEnvFloat`.

| Variable | Required | Default | Description |
|---|---|---|---|
| `TOPIC_ANALYSIS_MAX_TOKENS` | No | `2000` | Maximum tokens for topic analysis AI calls |
| `TOPIC_ANALYSIS_TEMPERATURE` | No | `0.2` | Temperature for topic analysis AI calls |
| `TOPIC_ANALYSIS_TIMEOUT_SECONDS` | No | `90` | Timeout in seconds for topic analysis AI calls |
| `TOPIC_ANALYSIS_RETRY_COUNT` | No | `3` | Maximum retries for topic analysis AI calls |

### Frontend (Nuxt)

These are set via `nuxt.config.ts` `runtimeConfig` and can be overridden with environment variables.

| Variable | Required | Default | Description |
|---|---|---|---|
| `API_INTERNAL_BASE` | No | `"http://localhost:5000/api"` | Server-side API base URL (used during SSR) |
| `NUXT_PUBLIC_API_ORIGIN` | No | `"http://localhost:5000"` | Public API origin exposed to the browser |
| `NUXT_PUBLIC_API_BASE` | No | `"http://localhost:5000/api"` | Public API base URL exposed to the browser |

### Docker Compose

These variables are used by the Docker Compose files and have no effect outside Docker.

| Variable | Required | Default | Description |
|---|---|---|---|
| `FRONT_PORT` | No | `"3001"` (SQLite compose), `"3000"` (internal) | Host port mapped to the frontend container |
| `BACKEND_PORT` | No | `"5000"` | Host port mapped to the backend container |
| `SQLITE_DB_FILE` | No | `"rss_reader.db"` | SQLite database filename (mounted volume) |
| `POSTGRES_DB` | No | `"rss_reader"` | PostgreSQL database name |
| `POSTGRES_USER` | No | `"postgres"` | PostgreSQL user |
| `POSTGRES_PASSWORD` | No | `"postgres"` | PostgreSQL password |
| `POSTGRES_PORT` | No | `"5432"` | Host port mapped to the PostgreSQL container |
| `TZ` | No | `"Asia/Shanghai"` | Timezone for PostgreSQL container |
| `GOPROXY` | No | *(empty)* | Go module proxy for backend build |
| `GOSUMDB` | No | *(empty)* | Go checksum database for backend build |
| `HTTP_PROXY` / `HTTPS_PROXY` / `NO_PROXY` | No | *(empty)* | Proxy settings forwarded to build contexts |

## Config File Format

The backend reads a YAML config file from `backend-go/configs/config.yaml`. This file is loaded via [Viper](https://github.com/spf13/viper) on startup. The shipped `config.yaml` contains a PostgreSQL example, but the code defaults are SQLite — the app works without the file at all.

```yaml
server:
  port: "5000"
  mode: "debug"           # debug | release | test

database:
  driver: "sqlite"        # sqlite | postgres
  dsn: "rss_reader.db"    # SQLite path or Postgres connection string
  sqlite:
    journal_mode: "WAL"
    busy_timeout_ms: 5000
    max_idle_conns: 2
    max_open_conns: 1
  postgres:
    max_idle_conns: 5
    max_open_conns: 25
    conn_max_lifetime_minutes: 60
    conn_max_idle_time_minutes: 10

cors:
  origins:
    - "http://localhost:3001"
    - "http://localhost:3000"
  methods:
    - "GET"
    - "POST"
    - "PUT"
    - "DELETE"
    - "OPTIONS"
  allow_headers:
    - "Content-Type"
    - "Authorization"
```

### Key Sections

- **server** — Controls the HTTP server port and Gin run mode. In `release` mode, Gin suppresses debug output.
- **database** — Configures the persistence layer. The `driver` field selects between SQLite and PostgreSQL. Each driver has its own tuning knobs for connection pooling.
- **cors** — Allowed origins, HTTP methods, and headers for cross-origin requests. Origins is a list; when overridden via `CORS_ORIGINS` env var, it is parsed as a comma-separated string.

## Required vs Optional Settings

All settings have defaults. The application will start without any configuration file or environment variables, using SQLite with sensible defaults.

No environment variable causes a startup failure if absent. The config loading code (`applyEnvOverrides` in `config.go`) only applies env values when they are non-empty; otherwise the YAML file or code defaults are used.

The only scenario that causes a startup failure is an invalid or unreachable database DSN — the `database.InitDB` call in `main.go` will `log.Fatalf` if the connection cannot be established.

## Defaults

### Backend Defaults

| Setting | Default | Source |
|---|---|---|
| Server port | `"5000"` | `viper.SetDefault` in `config.go` |
| Server mode | `"debug"` | `viper.SetDefault` in `config.go` |
| Database driver | `"sqlite"` | `viper.SetDefault` in `config.go` |
| Database DSN | `"rss_reader.db"` | `viper.SetDefault` in `config.go` |
| SQLite journal mode | `"WAL"` | `viper.SetDefault` in `config.go` |
| SQLite busy timeout | `5000` ms | `viper.SetDefault` in `config.go` |
| SQLite max idle conns | `2` | `viper.SetDefault` in `config.go` |
| SQLite max open conns | `1` | `viper.SetDefault` in `config.go` |
| Postgres max idle conns | `5` | `viper.SetDefault` in `config.go` |
| Postgres max open conns | `25` | `viper.SetDefault` in `config.go` |
| Postgres conn max lifetime | `60` min | `viper.SetDefault` in `config.go` |
| Postgres conn max idle time | `10` min | `viper.SetDefault` in `config.go` |
| CORS origins | `localhost:3001`, `localhost:3000` | `viper.SetDefault` in `config.go` |
| CORS methods | `GET, POST, PUT, DELETE, OPTIONS` | `viper.SetDefault` in `config.go` |
| CORS headers | `Content-Type, Authorization` | `viper.SetDefault` in `config.go` |
| Crawl service URL | `"http://localhost:11235"` | `runtime.go` fallback |
| Tracing enabled | `true` | `tracing.DefaultConfig()` |
| Tracing retention | `7` days | `tracing.DefaultConfig()` |
| Topic analysis max tokens | `2000` | `ai_analysis.go` `parseEnvInt` |
| Topic analysis temperature | `0.2` | `ai_analysis.go` `parseEnvFloat` |
| Topic analysis timeout | `90` s | `ai_analysis.go` `parseEnvInt` |
| Topic analysis retries | `3` | `ai_analysis.go` `parseEnvInt` |

### Frontend Defaults

| Setting | Default | Source |
|---|---|---|
| API internal base | `"http://localhost:5000/api"` | `nuxt.config.ts` |
| Public API origin | `"http://localhost:5000"` | `nuxt.config.ts` |
| Public API base | `"http://localhost:5000/api"` | `nuxt.config.ts` |

## Per-Environment Overrides

### Local Development

For local development, the defaults work out of the box:

- Backend runs on `http://localhost:5000` with SQLite.
- Frontend dev server (`pnpm dev`) runs on `http://localhost:3001`.
- No config file or `.env` file is required.

To switch the backend to PostgreSQL locally, create or edit `backend-go/configs/config.yaml` and set `database.driver: "postgres"` with the appropriate DSN.

### Docker (SQLite)

Use `docker-compose.sqlite.yml` for containerized deployment with SQLite persistence:

```bash
docker compose -f docker-compose.sqlite.yml up --build
```

The SQLite database file is persisted in `./data/` on the host.

### Docker (PostgreSQL + pgvector)

Use `docker-compose.yml` for PostgreSQL with the pgvector extension:

```bash
docker compose up -d
```

Set `POSTGRES_PASSWORD` and other Postgres variables in your environment or a `.env` file for production. The database is persisted in `./data/` on the host. An init script at `docker/postgres/init/01-enable-pgvector.sql` enables the pgvector extension on first start.

## Database-Stored Settings (AI Features)

AI-related configuration is not stored in files or environment variables — it is managed through the web UI and persisted in the `ai_settings` SQLite/Postgres table. The backend reads these at runtime via the `aisettings` package.

| Config Key | Description |
|---|---|
| `summary_config` | LLM credentials for article summarization (base URL, API key, model) |
| `auto_summary_config` | Auto-summary scheduler settings (time range, model params) |
| `firecrawl_config` | Firecrawl integration settings (enabled, API URL, API key, mode, timeout, max content length) |
| `open_notebook_config` | Open Notebook digest export settings (enabled, base URL, API key, model, target notebook, prompt mode, auto-send daily/weekly) |

These settings are loaded via `aisettings.LoadSummaryConfig()`, `aisettings.LoadFirecrawlConfig()`, etc. and are available through the Settings page in the frontend.
