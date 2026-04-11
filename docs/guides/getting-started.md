<!-- generated-by: gsd-doc-writer -->

# Getting Started

## Prerequisites

| Tool | Minimum Version | Notes |
|------|----------------|-------|
| [Node.js](https://nodejs.org/) | >= 18 | Required for the Nuxt 4 frontend |
| [pnpm](https://pnpm.io/) | >= 10 | Frontend package manager |
| [Go](https://go.dev/) | >= 1.25 | Required for the Gin backend |
| [Docker](https://www.docker.com/) | — | Optional, for containerized deployment |
| [Git](https://git-scm.com/) | — | For cloning the repository |
| [Python](https://www.python.org/) | >= 3.10 | Optional, for running integration tests in `tests/workflow/` |

No `.env` file is required for local development — the backend and frontend both have working defaults.

## Installation Steps

### 1. Clone the repository

```bash
git clone <repository-url>
cd my-robot
```

### 2. Start the backend

```bash
cd backend-go
go mod tidy
go run cmd/server/main.go
```

The backend starts on `http://localhost:5000` and creates a SQLite database file (`rss_reader.db`) in the working directory on first run.

### 3. Start the frontend

Open a new terminal:

```bash
cd front
pnpm install
pnpm dev
```

The frontend dev server starts on `http://localhost:3001`.

### 4. Verify the connection

Open `http://localhost:3001` in a browser. The frontend should load and connect to the backend at `http://localhost:5000/api`. You can begin adding RSS feeds immediately.

## Docker Compose (Alternative)

If you prefer containerized deployment, copy the environment template and start both services with a single command:

```bash
cp .env.example .env
docker compose -f docker-compose.sqlite.yml up --build
```

The `.env.example` file contains the minimal set of variables:

```
FRONT_PORT=3001
BACKEND_PORT=5000
SQLITE_DB_FILE=rss_reader.db
```

- Frontend: `http://localhost:3001`
- Backend: `http://localhost:5000`
- SQLite database persisted in `./data/rss_reader.db`

To use PostgreSQL with pgvector support instead:

```bash
docker compose -f docker-compose.yml up --build
```

Port mappings and other Docker settings can be customized via the `.env` file — see [Configuration](configuration.md) for the full list.

## First Run

Once both services are running:

1. Open `http://localhost:3001` in your browser.
2. Add an RSS feed via the subscription management panel.
3. The feed will be fetched and articles will appear in the three-pane reading layout.
4. (Optional) Configure AI features — LLM API key, Firecrawl, and digest settings — through the web UI Settings page. These are stored in the database and don't require config file edits.

## Common Setup Issues

### Port already in use

If `http://localhost:5000` or `http://localhost:3001` is occupied, set the ports via environment variables:

- Backend: set `SERVER_PORT` before running `go run cmd/server/main.go`.
- Frontend: set the `NUXT_PUBLIC_API_BASE` environment variable if the backend runs on a non-default port.
- Docker: set `FRONT_PORT` and `BACKEND_PORT` in `.env`.

### Backend fails to start with database errors

The backend defaults to SQLite. If the database file is corrupted, delete `rss_reader.db` (or the file specified by `DATABASE_DSN`) and restart — it will be recreated on startup.

### Frontend cannot connect to backend

Ensure the backend is running on `http://localhost:5000`. The frontend API base URL defaults to `http://localhost:5000/api` and can be overridden with the `NUXT_PUBLIC_API_BASE` environment variable if needed.

### Go module download failures (China region)

If `go mod tidy` is slow or fails, set a Go module proxy:

```bash
go env -w GOPROXY=https://goproxy.cn,direct
```

Similarly for pnpm, you can set a registry:

```bash
pnpm config set registry https://registry.npmmirror.com
```

### Docker build proxy settings

For Docker builds behind a proxy, configure `GOPROXY`, `NPM_CONFIG_REGISTRY`, `HTTP_PROXY`, and `HTTPS_PROXY` in your `.env` file. These are forwarded to the build contexts.

## Next Steps

- **[Configuration](configuration.md)** — Full list of environment variables, config file options, and database-stored AI settings.
- **[Development Guide](../operations/development.md)** — Build commands, test commands, coding conventions, and submission checklist.
- **[Architecture Overview](../architecture/overview.md)** — System design, component relationships, data flow, and background scheduler details.
