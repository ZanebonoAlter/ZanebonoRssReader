# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is an RSS Reader application with a Python Flask backend and Nuxt 4 frontend. The app features AI-powered article summarization, automatic feed refreshing, and a FeedBro-style three-column layout (sidebar navigation, article list, article content reader). Key features include categorizing feeds, marking articles as read/favorite, OPML import/export, and intelligent AI analysis.

**Note**: This application does NOT include user authentication - it's designed for personal/single-user deployment.

## Architecture

### Backend (Python Flask)
- **Entry point**: [backend/app.py](backend/app.py) - Flask application factory pattern with blueprint registration
- **Configuration**: [backend/config.py](backend/config.py) - Application configuration including CORS origins
- **Database**: SQLite with [backend/database.py](backend/database.py) initialization
- **Models**: [backend/models.py](backend/models.py) - SQLAlchemy ORM models:
  - `Category` - RSS feed categories
  - `Feed` - RSS feed sources with auto-refresh settings
  - `Article` - Parsed articles from feeds
  - `AISummary` - AI-generated category summaries
  - `SchedulerTask` - Background task execution tracking
  - `AISettings` - AI service configuration storage
- **RSS Parser**: [backend/rss_parser.py](backend/rss_parser.py) - RSS/Atom feed parsing with feedparser
- **Scheduling System**:
  - [backend/scheduler_base.py](backend/scheduler_base.py) - Base scheduler class with execution tracking
  - [backend/auto_refresh.py](backend/auto_refresh.py) - Automatic feed refresh scheduler
  - [backend/auto_summary.py](backend/auto_summary.py) - Automatic AI summary generation
- **Background Tasks**: [backend/tasks.py](backend/tasks.py) - Task management utilities
- **Data Initialization**: [backend/init_data.py](backend/init_data.py) - Seed data for development
- **Routes**: Modular blueprint architecture in [backend/routes/](backend/routes/):
  - `categories.py` - `/api/categories` - Category CRUD operations
  - `feeds.py` - `/api/feeds` - Feed management, preview, manual refresh
  - `articles.py` - `/api/articles` - Article retrieval, filtering, read/favorite status
  - `ai.py` - `/api/ai` - AI article summarization, settings management, connection testing
  - `summaries.py` - `/api/summaries` - AI summary CRUD, generation, retrieval
  - `schedulers.py` - `/api/schedulers` - Background task control, status monitoring
  - `opml.py` - `/api/opml` - OPML import/export functionality

### Frontend (Nuxt 4)
- **Config**: [front/nuxt.config.ts](front/nuxt.config.ts) - Nuxt configuration with Tailwind CSS, Pinia, VueUse
- **Root Component**: [front/app/app.vue](front/app/app.vue) - Main application entry point
- **Pages**: File-based routing in [front/app/pages/](front/app/pages/)
  - `index.vue` - Main feed reader page with FeedLayout
  - `article/[id].vue` - Article detail page
- **Components**: [front/app/components/](front/app/components/)
  - `FeedLayout.vue` - Three-column layout (sidebar, article list, content)
  - `Crud/` - CRUD dialog components (Add/Edit Category, Add/Edit Feed)
  - `summary/` - AI summary components (AISummary, AISummaryDetail, AISummariesList)
  - `ArticleCard.vue`, `CategoryCard.vue`, `FeedIcon.vue` - UI components
  - `GlobalSettingsDialog.vue`, `ImportOpmlDialog.vue` - Settings dialogs
- **Composables**: [front/app/composables/](front/app/composables/)
  - `useApi.ts` - Centralized API client with typed interfaces
  - `useAI.ts` - AI summarization and analysis functions
  - `useAutoRefresh.ts` - Auto-refresh polling and scheduling
  - `useRssParser.ts` - RSS feed parsing utilities
- **State Management**: [front/app/stores/](front/app/stores/) - Pinia stores
  - `api.ts` - Main API store that fetches from backend
  - `feeds.ts` - Local feeds/categories state
  - `articles.ts` - Local articles state
- **Types**: [front/app/types/index.ts](front/app/types/index.ts) - TypeScript type definitions
- **Plugins**: [front/app/plugins/](front/app/plugins/) - Nuxt plugins (dayjs, etc.)

## Development Commands

### Backend

#### Using UV (Recommended)
```bash
cd backend
# Sync dependencies and create virtual environment
uv sync

# Activate virtual environment
# Windows:
.venv\Scripts\activate
# macOS/Linux:
source .venv/bin/activate

# Run backend (port 5000)
python app.py
```

#### Using Traditional pip
```bash
cd backend
# Create virtual environment (first time only)
python -m venv venv
# Windows activation:
venv\Scripts\activate
# macOS/Linux activation:
source venv/bin/activate

# Install dependencies
pip install -r requirements.txt

# Run backend (port 5000)
python app.py
```

### Frontend
```bash
cd front
# Install dependencies
pnpm install

# Run dev server (port 3001)
pnpm dev

# Build for production
pnpm build
```

### Quick Start (Both)
```bash
# From project root - starts both backend and frontend
start-all.bat
```

## Key Integration Points

1. **API Base URL**: Frontend connects to `http://localhost:5000/api` (defined in [front/app/composables/useApi.ts](front/app/composables/useApi.ts:1))

2. **Data Flow**: Backend → `useApiStore.fetchX()` → local stores (`useFeedsStore`, `useArticlesStore`) → components

3. **ID Types**: Backend uses integer IDs, frontend stores convert to strings. API methods expect numbers for backend calls.

4. **RSS Parsing**: Backend uses `feedparser` library in [backend/rss_parser.py](backend/rss_parser.py) to fetch and parse RSS/Atom feeds

## Database Models

**Category**: id, name, slug, icon, color, description, created_at
**Feed**: id, title, description, url, category_id (FK), icon, color, last_updated, created_at, max_articles, refresh_interval, refresh_status, refresh_error, last_refresh_at, ai_summary_enabled
**Article**: id, feed_id (FK), title, description, content, link, pub_date, author, read, favorite, created_at
**AISummary**: id, category_id, title, summary, key_points, articles, article_count, time_range, created_at, updated_at
**SchedulerTask**: id, name, description, check_interval, last_execution_time, next_execution_time, status, last_error, last_error_time, total_executions, successful_executions, failed_executions, consecutive_failures, last_execution_duration, last_execution_result, created_at, updated_at
**AISettings**: id, key, value, description, created_at, updated_at

Cascading deletes are enabled: deleting a category deletes its feeds; deleting a feed deletes its articles.

## AI Features

The application includes AI-powered article analysis:

1. **Article Summarization**: POST `/api/ai/summarize`
   - Accepts: article title, content, AI credentials (base_url, api_key, model)
   - Returns: one-sentence summary, key points, takeaways, tags
   - Supports both Chinese and English

2. **AI Settings Management**: GET/POST `/api/ai/settings`
   - Store AI configuration in database
   - Supports any OpenAI-compatible API

3. **Category Summaries**: GET/POST `/api/summaries`
   - Generate aggregated summaries for multiple articles in a category
   - Configurable time ranges

4. **Scheduler Integration**: Background tasks can automatically generate summaries
   - Auto-refresh scheduler updates feeds
   - Auto-summary scheduler generates AI summaries

## Python Environment

- **Runtime**: Python 3.13+ (specified in [pyproject.toml](pyproject.toml:6))
- **Dependency Manager**: uv (lockfile at [uv.lock](uv.lock))
- **Core dependencies**: Flask 3.0, SQLAlchemy, feedparser, Flask-CORS, requests, beautifulsoup4, crawl4ai
