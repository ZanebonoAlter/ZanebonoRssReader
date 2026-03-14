# Backend Agent Guide

**Scope:** `backend-go/` — Go + Gin + GORM + SQLite backend

## Overview
RSS Reader backend using domain-driven design with clear separation between HTTP handlers and business logic.

## Structure
```
backend-go/
├── cmd/
│   ├── server/        # Main entry point
│   ├── migrate-digest/# Migration tool
│   └── test-digest/   # Test harness
├── internal/
│   ├── app/           # HTTP routing, middleware
│   ├── domain/        # Business logic by domain
│   │   ├── models/    # GORM models
│   │   ├── feeds/
│   │   ├── articles/
│   │   ├── summaries/
│   │   ├── digest/
│   │   └── ...
│   ├── jobs/          # Background job processors
│   └── platform/      # Shared infrastructure
└── go.mod
```

## Where to Look
| Task | Location |
|------|----------|
| HTTP routes | `internal/app/router.go` |
| Request handlers | `internal/domain/*/handlers.go` |
| Business logic | `internal/domain/*/` |
| Data models | `internal/domain/models/` |
| Background jobs | `internal/jobs/` |
| Shared infra | `internal/platform/` |

## Conventions

### Handler Pattern
```go
func GetArticles(c *gin.Context) {
    // Validate params first
    feedID := c.Query("feed_id")
    if feedID == "" {
        c.JSON(400, gin.H{"success": false, "error": "feed_id required"})
        return
    }
    
    // Business logic
    articles, err := articleService.List(feedID)
    if err != nil {
        c.JSON(500, gin.H{"success": false, "error": err.Error()})
        return
    }
    
    c.JSON(200, gin.H{"success": true, "data": articles})
}
```

### Error Handling
```go
// Wrap errors with context
if err := db.Save(&article).Error; err != nil {
    return fmt.Errorf("failed to save article %d: %w", article.ID, err)
}
```

### JSON Tags
Always use snake_case in struct tags:
```go
type Article struct {
    ID        uint      `json:"id"`
    FeedID    uint      `json:"feed_id"`     // snake_case
    CreatedAt time.Time `json:"created_at"`  // snake_case
}
```

### Imports
```go
import (
    "fmt"
    "time"
    
    "github.com/gin-gonic/gin"
    "gorm.io/gorm"
    
    "myrobot/internal/domain/models"
    "myrobot/internal/platform/database"
)
```

### Naming
- Exported: `PascalCase` (e.g., `ArticleService`)
- Private: `lowerCamelCase` (e.g., `validateFeed`)
- Short package names (e.g., `models`, `db`)

## Anti-Patterns
- DON'T put business logic in `router.go`
- DON'T ignore errors (always handle or wrap)
- DON'T use panic for error handling
- DON'T access DB directly from handlers (use domain layer)

## Commands
```bash
cd backend-go
go mod tidy
go run cmd/server/main.go     # http://localhost:5000
go test ./...
go build ./...

# Single package
go test ./internal/domain/feeds -v
```

## Notes
- SQLite for persistence (single-user app)
- WebSocket at `ws://localhost:5000/ws`
- API base: `http://localhost:5000/api`
- Use `gofmt` for formatting
- Table-driven tests preferred
