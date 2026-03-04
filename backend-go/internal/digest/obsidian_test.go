package digest

import (
	"my-robot-backend/internal/models"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestObsidianExporter(t *testing.T) {
	tempDir := t.TempDir()
	exporter := NewObsidianExporter(tempDir)

	date := time.Date(2026, 3, 4, 0, 0, 0, 0, time.Local)
	digests := []CategoryDigest{
		{
			CategoryName: "AI技术",
			CategoryID:   1,
			FeedCount:    2,
			AISummaries: []models.AISummary{
				{
					ID:      1,
					Title:   "Test 1",
					Summary: "Content 1",
					Feed: &models.Feed{
						Title: "Tech Feed",
					},
				},
			},
		},
	}

	err := exporter.ExportDailyDigest(date, digests)
	assert.NoError(t, err)

	expectedPath := filepath.Join(tempDir, "Daily", "AI技术", "2026-03-04-日报.md")
	_, err = os.Stat(expectedPath)
	assert.NoError(t, err, "Daily digest file should be created")

	content, err := os.ReadFile(expectedPath)
	assert.NoError(t, err)
	assert.Contains(t, string(content), "category: AI技术")
	assert.Contains(t, string(content), "date: 2026-03-04")
	assert.Contains(t, string(content), "type: daily-digest")
	assert.Contains(t, string(content), "AI技术 - 2026年3月4日日报")
	assert.Contains(t, string(content), "Tech Feed")
	assert.Contains(t, string(content), "Content 1")
}

func TestObsidianExporterWeekly(t *testing.T) {
	tempDir := t.TempDir()
	exporter := NewObsidianExporter(tempDir)

	weekStart := time.Date(2026, 3, 1, 0, 0, 0, 0, time.Local)
	digests := []CategoryDigest{
		{
			CategoryName: "AI技术",
			CategoryID:   1,
			FeedCount:    3,
			AISummaries: []models.AISummary{
				{
					ID:      1,
					Title:   "Test 1",
					Summary: "Weekly summary 1",
					Feed: &models.Feed{
						Title: "Weekly Feed",
					},
				},
			},
		},
	}

	err := exporter.ExportWeeklyDigest(weekStart, digests)
	assert.NoError(t, err)

	expectedPath := filepath.Join(tempDir, "Weekly", "AI技术", "2026-W9-周报.md")
	_, err = os.Stat(expectedPath)
	assert.NoError(t, err, "Weekly digest file should be created")

	content, err := os.ReadFile(expectedPath)
	assert.NoError(t, err)
	assert.Contains(t, string(content), "category: AI技术")
	assert.Contains(t, string(content), "week: 2026-W9")
	assert.Contains(t, string(content), "type: weekly-digest")
	assert.Contains(t, string(content), "AI技术 - 第9周周报")
	assert.Contains(t, string(content), "Weekly summary 1")
}

func TestObsidianExporterMultipleCategories(t *testing.T) {
	tempDir := t.TempDir()
	exporter := NewObsidianExporter(tempDir)

	date := time.Date(2026, 3, 4, 0, 0, 0, 0, time.Local)
	digests := []CategoryDigest{
		{
			CategoryName: "AI技术",
			CategoryID:   1,
			FeedCount:    1,
			AISummaries: []models.AISummary{
				{ID: 1, Title: "Test 1", Summary: "Content 1"},
			},
		},
		{
			CategoryName: "开发工具",
			CategoryID:   2,
			FeedCount:    1,
			AISummaries: []models.AISummary{
				{ID: 2, Title: "Test 2", Summary: "Content 2"},
			},
		},
	}

	err := exporter.ExportDailyDigest(date, digests)
	assert.NoError(t, err)

	path1 := filepath.Join(tempDir, "Daily", "AI技术", "2026-03-04-日报.md")
	path2 := filepath.Join(tempDir, "Daily", "开发工具", "2026-03-04-日报.md")

	_, err = os.Stat(path1)
	assert.NoError(t, err)
	_, err = os.Stat(path2)
	assert.NoError(t, err)
}

func TestObsidianExporterUnknownFeed(t *testing.T) {
	tempDir := t.TempDir()
	exporter := NewObsidianExporter(tempDir)

	date := time.Date(2026, 3, 4, 0, 0, 0, 0, time.Local)
	digests := []CategoryDigest{
		{
			CategoryName: "AI技术",
			CategoryID:   1,
			FeedCount:    1,
			AISummaries: []models.AISummary{
				{
					ID:      1,
					Title:   "Test 1",
					Summary: "Content without feed",
					Feed:    nil,
				},
			},
		},
	}

	err := exporter.ExportDailyDigest(date, digests)
	assert.NoError(t, err)

	expectedPath := filepath.Join(tempDir, "Daily", "AI技术", "2026-03-04-日报.md")
	content, err := os.ReadFile(expectedPath)
	assert.NoError(t, err)
	assert.Contains(t, string(content), "### 未知订阅源")
	assert.Contains(t, string(content), "Content without feed")
}

func TestGetWeekNumber(t *testing.T) {
	exporter := NewObsidianExporter("")

	t1 := time.Date(2026, 3, 4, 0, 0, 0, 0, time.Local)
	assert.Equal(t, 10, exporter.getWeekNumber(t1))

	t2 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local)
	weekNum := exporter.getWeekNumber(t2)
	assert.Greater(t, weekNum, 0)
	assert.LessOrEqual(t, weekNum, 53)
}
