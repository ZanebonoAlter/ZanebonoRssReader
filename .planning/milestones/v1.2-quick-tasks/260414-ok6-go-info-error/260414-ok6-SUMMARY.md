# Quick Task 260414-ok6 Summary

## Goal

做后端 Go 日志管理，将 info 与 error 输出区分开，减少日志混杂。

## What Changed

- 新增 `backend-go/internal/platform/logging/logging.go`，提供轻量日志门面：`Infof/Infoln/Warnf/Warnln/Errorf/Errorln/Fatalf`。
- `ConfigureStdlib()` 现在会把标准库 `log` 输出按内容路由：常规日志走 `stdout`，`warning/error/fatal/panic/failed` 相关日志走 `stderr`。
- `cmd/server/main.go` 与 `internal/app/runtime.go` 改为显式使用日志门面，服务启动、初始化、优雅退出阶段的日志级别更清晰。
- `auto_refresh`、`auto_tag_merge`、`content_completion` 三个调度器改为使用日志门面，运行信息和异常输出不再混在一起。
- 更新 `docs/architecture/backend-go.md` 与 `docs/operations/development.md`，补充日志门面与 stdout/stderr 分流说明。

## Scope Notes

- 工作区里已有其他未提交改动正落在 `auto_summary`、`summary_queue`、`tagger` 等文件上。
- 为避免把别人的业务改动一起提交，本次 quick 任务没有继续提交这些重叠文件上的日志替换。
- 即便如此，标准库 `log` 已经全局分流，启动链和主要调度器的日志混杂问题已经明显改善。

## Verification

- `cd backend-go && go test ./internal/platform/logging -v`
- `cd backend-go && go test ./internal/jobs -run "TestAutoRefresh|TestContentCompletion" -v`
- `cd backend-go && go build ./...`

## Commits

- `ad1a81e` `feat(logging): split startup logs by severity`
- `3b5ca38` `feat(logging): route scheduler errors to stderr`
