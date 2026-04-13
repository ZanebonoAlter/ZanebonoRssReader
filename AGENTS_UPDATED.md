AGENTS.md has been successfully updated with the latest project information:

## Key Changes Made:
1. **Project Snapshot Updated**:
   - Main branch now uses PostgreSQL with pgvector for vector search, SQLite is only in the `sqlite` archive branch
   - Added optional Redis support for persistent job queues
   - Added crawl service (default port 11235) for content completion
   - Mentioned AI configuration is stored in the database and managed via web UI

2. **Build/Test Section Updated**:
   - Added Playwright E2E test commands for frontend (`pnpm test:e2e`, `pnpm test:e2e:ui`)
   - Added PostgreSQL Docker run command for local development setup
   - Added backend auxiliary migration commands (migrate-tags, migrate-db)

3. **Conventions Updated**:
   - Clarified frontend `pages/` directory should be thin (only component mounting, no business logic)
   - Added more domain packages to backend conventions (topicanalysis, topicgraph, preferences)
   - Explicitly stated business logic belongs in domain packages, not in handlers/job processors

4. **Added Recommended Pre-push Verification Sequence**:
   ```bash
   cd backend-go && go test ./... && go build ./...
   cd front && pnpm exec nuxi typecheck && pnpm test:unit && pnpm build
   ```

All changes are based on the latest docs (configuration.md, testing.md, development.md) and current project structure.