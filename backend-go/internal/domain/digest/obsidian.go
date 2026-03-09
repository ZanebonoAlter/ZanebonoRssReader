package digest

import (
	"fmt"
	"my-robot-backend/internal/domain/models"
	"os"
	"path/filepath"
	"time"
)

type ObsidianExporter struct {
	vaultPath string
}

func NewObsidianExporter(vaultPath string) *ObsidianExporter {
	return &ObsidianExporter{
		vaultPath: vaultPath,
	}
}

func (e *ObsidianExporter) ExportDailyDigest(date time.Time, digests []CategoryDigest) error {
	for _, digest := range digests {
		if err := e.exportCategoryDaily(date, digest); err != nil {
			return err
		}
	}
	return nil
}

func (e *ObsidianExporter) ExportWeeklyDigest(weekStart time.Time, digests []CategoryDigest) error {
	weekNum := e.getWeekNumber(weekStart)
	for _, digest := range digests {
		if err := e.exportCategoryWeekly(weekStart, weekNum, digest); err != nil {
			return err
		}
	}
	return nil
}

func (e *ObsidianExporter) exportCategoryDaily(date time.Time, digest CategoryDigest) error {
	dirPath := filepath.Join(e.vaultPath, "Daily", digest.CategoryName)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return err
	}

	filename := fmt.Sprintf("%s-日报.md", date.Format("2006-01-02"))
	filePath := filepath.Join(dirPath, filename)

	content := e.generateDailyMarkdown(date, digest)

	return os.WriteFile(filePath, []byte(content), 0644)
}

func (e *ObsidianExporter) exportCategoryWeekly(weekStart time.Time, weekNum int, digest CategoryDigest) error {
	dirPath := filepath.Join(e.vaultPath, "Weekly", digest.CategoryName)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return err
	}

	filename := fmt.Sprintf("%d-W%d-周报.md", weekStart.Year(), weekNum)
	filePath := filepath.Join(dirPath, filename)

	weekEnd := weekStart.AddDate(0, 0, 6)
	content := e.generateWeeklyMarkdown(weekStart, weekEnd, digest)

	return os.WriteFile(filePath, []byte(content), 0644)
}

func (e *ObsidianExporter) generateDailyMarkdown(date time.Time, digest CategoryDigest) string {
	return fmt.Sprintf(`---
category: %s
date: %s
type: daily-digest
tags: [daily, %s, %s]
---

# %s - %s日报

## 本日概要

本日共生成%d份订阅源总结。

## 订阅源总结

%s
`,
		digest.CategoryName,
		date.Format("2006-01-02"),
		digest.CategoryName,
		date.Format("2006-01"),
		digest.CategoryName,
		date.Format("2006年1月2日"),
		digest.FeedCount,
		e.generateSummariesSection(digest.AISummaries),
	)
}

func (e *ObsidianExporter) generateWeeklyMarkdown(weekStart, weekEnd time.Time, digest CategoryDigest) string {
	weekNum := e.getWeekNumber(weekStart)
	return fmt.Sprintf(`---
category: %s
week: %d-W%d
date_range: %s ~ %s
type: weekly-digest
tags: [weekly, %s]
---

# %s - 第%d周周报 (%s-%s)

## 本周概览

本周共生成%d份订阅源总结。

## 按订阅源汇总

%s
`,
		digest.CategoryName,
		weekStart.Year(),
		weekNum,
		weekStart.Format("2006-01-02"),
		weekEnd.Format("2006-01-02"),
		digest.CategoryName,
		digest.CategoryName,
		weekNum,
		weekStart.Format("1.2"),
		weekEnd.Format("1.2"),
		digest.FeedCount,
		e.generateSummariesSection(digest.AISummaries),
	)
}

func (e *ObsidianExporter) generateSummariesSection(summaries []models.AISummary) string {
	var content string
	for _, summary := range summaries {
		feedName := "未知订阅源"
		if summary.Feed != nil {
			feedName = summary.Feed.Title
		}
		content += fmt.Sprintf("### %s\n\n%s\n\n", feedName, summary.Summary)
	}
	return content
}

func (e *ObsidianExporter) getWeekNumber(date time.Time) int {
	_, week := date.ISOWeek()
	return week
}
