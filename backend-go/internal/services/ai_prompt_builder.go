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

type PromptPreferenceContext struct {
	Personalized  bool
	FeedCount     int
	CategoryCount int
}

func NewAISummaryPromptBuilder(prefService *PreferenceService, db *gorm.DB) *AISummaryPromptBuilder {
	return &AISummaryPromptBuilder{
		preferenceService: prefService,
		db:                db,
	}
}

func (b *AISummaryPromptBuilder) BuildPersonalizedPrompt(
	feedName string,
	categoryName string,
	articlesText string,
	articleCount int,
	language string,
) (string, PromptPreferenceContext, error) {
	var userContext strings.Builder
	context := PromptPreferenceContext{}

	if language == "zh" {
		userContext.WriteString("## 用户偏好背景\n\n")
	} else {
		userContext.WriteString("## User Preference Context\n\n")
	}

	topFeeds, err := b.preferenceService.GetTopPreferredFeeds(5)
	if err == nil && len(topFeeds) > 0 {
		context.FeedCount = len(topFeeds)
		if language == "zh" {
			userContext.WriteString("### 偏好订阅源\n")
		} else {
			userContext.WriteString("### Preferred Feeds\n")
		}
		for _, feedID := range topFeeds {
			var feed struct {
				Title string
			}
			if err := b.db.Table("feeds").
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
		context.CategoryCount = len(topCategories)
		if language == "zh" {
			userContext.WriteString("### 偏好分类\n")
		} else {
			userContext.WriteString("### Preferred Categories\n")
		}
		for _, catID := range topCategories {
			var category struct {
				Name string
			}
			if err := b.db.Table("categories").
				Select("name").
				Where("id = ?", catID).
				First(&category).Error; err == nil {
				userContext.WriteString(fmt.Sprintf("- %s\n", category.Name))
			}
		}
		userContext.WriteString("\n")
	}

	type Stats struct {
		PreferenceCount int
		AvgReadingTime  int
		AvgScrollDepth  float64
	}

	var stats Stats
	if err := b.db.Raw(`
		SELECT
			COUNT(*) as preference_count,
			COALESCE(AVG(avg_reading_time), 0) as avg_reading_time,
			COALESCE(AVG(scroll_depth_avg), 0) as avg_scroll_depth
		FROM user_preferences
	`).Scan(&stats).Error; err == nil && stats.PreferenceCount > 0 {
		context.Personalized = true
		if language == "zh" {
			userContext.WriteString("### 阅读习惯\n")
			readingStyle := "中等深度"
			if stats.AvgReadingTime > 180 {
				readingStyle = "偏爱细节"
			} else if stats.AvgReadingTime < 60 {
				readingStyle = "偏向快读"
			}
			userContext.WriteString(fmt.Sprintf("- 平均阅读时长: %d 秒（%s）\n", stats.AvgReadingTime, readingStyle))

			attentionLevel := "中等"
			if stats.AvgScrollDepth > 80 {
				attentionLevel = "高，通常会读完"
			} else if stats.AvgScrollDepth < 40 {
				attentionLevel = "低，更像快速扫读"
			}
			userContext.WriteString(fmt.Sprintf("- 平均滚动深度: %.0f%%（%s）\n\n", stats.AvgScrollDepth, attentionLevel))
		} else {
			userContext.WriteString("### Reading Habits\n")
			readingStyle := "balanced"
			if stats.AvgReadingTime > 180 {
				readingStyle = "detail-oriented"
			} else if stats.AvgReadingTime < 60 {
				readingStyle = "quick scan"
			}
			userContext.WriteString(fmt.Sprintf("- Average reading time: %d seconds (%s)\n", stats.AvgReadingTime, readingStyle))

			attentionLevel := "moderate"
			if stats.AvgScrollDepth > 80 {
				attentionLevel = "high attention"
			} else if stats.AvgScrollDepth < 40 {
				attentionLevel = "light attention"
			}
			userContext.WriteString(fmt.Sprintf("- Average scroll depth: %.0f%% (%s)\n\n", stats.AvgScrollDepth, attentionLevel))
		}
	}

	context.Personalized = context.Personalized || context.FeedCount > 0 || context.CategoryCount > 0
	basePrompt := b.buildBasePrompt(feedName, categoryName, articlesText, articleCount, language)
	if !context.Personalized {
		return basePrompt, context, nil
	}

	userContext.WriteString("---\n\n")
	return userContext.String() + basePrompt, context, nil
}

func (b *AISummaryPromptBuilder) buildBasePrompt(
	feedName string,
	categoryName string,
	articlesText string,
	articleCount int,
	language string,
) string {
	if language == "zh" {
		target := fmt.Sprintf("订阅源“%s”", feedName)
		if categoryName != "" {
			target += fmt.Sprintf("（分类：%s）", categoryName)
		}

		return fmt.Sprintf(`请总结以下来自%s的 %d 篇文章。

文章列表（按时间倒序）：
%s

请按下面格式输出：

## 核心主题
用一句话概括这批文章的共同主题和趋势。

## 重要新闻

### 热点事件
列出 2-3 个最重要的事件。每个事件包含：
- 加粗标题
- 2-3 句简述
- 引用来源，格式为 > [文章标题](链接)

### 其他重要新闻
列出其余值得关注的新闻。每条包含：
- 加粗标题
- 1-2 句简述
- 引用来源

## 核心观点
总结 3-5 个关键信号或趋势。

## 相关标签
#%s #标签1 #标签2 #标签3

注意：
1. 每条新闻都必须带来源引用。
2. 引用格式固定为 > [文章标题](文章链接)。
3. 总结要短，重点要清楚。
4. 保持客观、中立。
5. 可以根据用户偏好调整详略和关注点。`, target, articleCount, articlesText, feedName)
	}

	target := fmt.Sprintf("feed \"%s\"", feedName)
	if categoryName != "" {
		target += fmt.Sprintf(" (category: %s)", categoryName)
	}

	return fmt.Sprintf(`Please summarize the following %d articles from %s.

Articles (in reverse chronological order):
%s

Please provide a summary in the following format:

## Core Theme
One sentence summarizing the core theme and trends.

## Important News

### Hot Topics
List 2-3 most important events, each including:
- Event title (in bold)
- Brief description (2-3 sentences)
- Source citation using > [Article Title](link)

### Other Important News
List other important news, each including:
- News title (in bold)
- Brief description (1-2 sentences)
- Source citation

## Key Points
Summarize 3-5 key points or trends.

## Related Tags
#%s #tag1 #tag2 #tag3

Important Notes:
1. Always cite news sources.
2. Source format: > [Article Title](article link).
3. Keep the summary concise and focused.
4. Maintain an objective and neutral tone.
5. Adjust content depth based on user preferences.`, articleCount, target, articlesText, feedName)
}
