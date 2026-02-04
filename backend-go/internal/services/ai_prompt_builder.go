package services

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

type AISummaryPromptBuilder struct {
	preferenceService *PreferenceService
	db                *gorm.DB
}

func NewAISummaryPromptBuilder(prefService *PreferenceService, db *gorm.DB) *AISummaryPromptBuilder {
	return &AISummaryPromptBuilder{
		preferenceService: prefService,
		db:                db,
	}
}

func (b *AISummaryPromptBuilder) BuildPersonalizedPrompt(
	categoryName string,
	articlesText string,
	articleCount int,
	language string,
) (string, error) {
	var userContext strings.Builder

	userContext.WriteString("## 用户偏好背景\n\n")

	topFeeds, err := b.preferenceService.GetTopPreferredFeeds(5)
	if err == nil && len(topFeeds) > 0 {
		userContext.WriteString("### 偏好订阅源\n")
		for _, feedID := range topFeeds {
			var feed struct {
				Title string
			}
			if err := b.preferenceService.db.Table("feeds").
				Select("title").
				Where("id = ?", feedID).
				First(&feed).Error; err == nil {
				userContext.WriteString(fmt.Sprintf("- %s\n", feed.Title))
			}
		}
		userContext.WriteString("\n")
	}

	topCategories, err := b.preferenceService.GetTopPreferredCategories(3)
	if err == nil && len(topCategories) > 0 {
		userContext.WriteString("### 偏好分类\n")
		for _, catID := range topCategories {
			var category struct {
				Name string
			}
			if err := b.preferenceService.db.Table("categories").
				Select("name").
				Where("id = ?", catID).
				First(&category).Error; err == nil {
				userContext.WriteString(fmt.Sprintf("- %s\n", category.Name))
			}
		}
		userContext.WriteString("\n")
	}

	type Stats struct {
		AvgReadingTime int
		AvgScrollDepth float64
	}
	var stats Stats
	if err := b.preferenceService.db.Raw(`
		SELECT 
			COALESCE(AVG(avg_reading_time), 0) as avg_reading_time,
			COALESCE(AVG(scroll_depth_avg), 0) as avg_scroll_depth
		FROM user_preferences
	`).Scan(&stats).Error; err == nil {
		userContext.WriteString("### 阅读习惯\n")

		readingStyle := "中等"
		if stats.AvgReadingTime > 180 {
			readingStyle = "详细深入"
		} else if stats.AvgReadingTime < 60 {
			readingStyle = "快速浏览"
		}
		userContext.WriteString(fmt.Sprintf("- 平均阅读时长: %d 秒 (%s)\n", stats.AvgReadingTime, readingStyle))

		attentionLevel := "中等"
		if stats.AvgScrollDepth > 80 {
			attentionLevel = "高 (完整阅读)"
		} else if stats.AvgScrollDepth < 40 {
			attentionLevel = "低 (快速浏览)"
		}
		userContext.WriteString(fmt.Sprintf("- 平均滚动深度: %.0f%% (%s)\n", stats.AvgScrollDepth, attentionLevel))
		userContext.WriteString("\n")
	}

	userContext.WriteString("---\n\n")

	basePrompt := b.buildBasePrompt(categoryName, articlesText, articleCount, language)

	return userContext.String() + basePrompt, nil
}

func (b *AISummaryPromptBuilder) buildBasePrompt(
	categoryName string,
	articlesText string,
	articleCount int,
	language string,
) string {
	if language == "zh" {
		return fmt.Sprintf(`请对以下来自"%s"分类的 %d 篇文章进行汇总总结。

文章列表（按时间倒序）：
%s

请提供以下格式的总结：

## 核心主题
用一句话概括这批文章的核心主题和趋势。

## 重要新闻

### 🔥 热点事件
列出2-3个最重要的事件，每个事件包含：
- 事件标题（用加粗）
- 简要说明（2-3句话）
- 引文标注新闻来源（使用 > [来源名称](链接) 格式）

### 📰 其他重要新闻
列出其他重要新闻，每条包含：
- 新闻标题（用加粗）
- 简要说明（1-2句话）
- 引文标注新闻来源（使用 > [来源名称](链接) 格式）

## 核心观点
总结3-5个核心观点或趋势，每个观点用简洁的语言表达。

## 相关标签
#标签1 #标签2 #标签3

**重要提醒**：
1. 必须为每条新闻标注来源，使用引文格式
2. 来源格式：> [来源订阅源名称](文章链接)
3. 确保总结简洁明了，突出重点
4. 保持客观中立的语气
5. 根据用户偏好调整内容的深度和重点`, categoryName, articleCount, articlesText)
	}

	return fmt.Sprintf(`Please summarize the following %d articles from "%s" category.

Articles (in reverse chronological order):
%s

Please provide a summary in the following format:

## Core Theme
One sentence summarizing the core theme and trends.

## Important News

### 🔥 Hot Topics
List 2-3 most important events, each including:
- Event title (in bold)
- Brief description (2-3 sentences)
- Source citation (using > [Source Name](link) format)

### 📰 Other Important News
List other important news, each including:
- News title (in bold)
- Brief description (1-2 sentences)
- Source citation

## Key Points
Summarize 3-5 key points or trends.

## Related Tags
#tag1 #tag2 #tag3

**Important Notes**:
1. Always cite news sources
2. Source format: > [Source Feed Name](article link)
3. Keep summary concise and focused
4. Maintain objective and neutral tone
5. Adjust content depth based on user preferences`, articleCount, categoryName, articlesText)
}
