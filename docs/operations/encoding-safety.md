# Encoding Safety

## Problem

On Windows, shell-based rewrites can corrupt non-ASCII files.

## High-risk operations

- `Set-Content`
- `Out-File`
- `>` or `>>`
- `git show > file`
- bulk rewrites of `.vue`, `.ts`, `.go`, `.md`

## Rules

1. Prefer UTF-8.
2. Re-open edited files after writing.
3. Search for suspicious text after large rewrites.
4. If corruption appears, rewrite the whole file cleanly instead of patching mojibake.

## Incident references

- `front/app/components/article/ArticleContent.vue`
- `backend-go/internal/schedulers/auto_summary.go`
- `backend-go/internal/schedulers/content_completion.go`
