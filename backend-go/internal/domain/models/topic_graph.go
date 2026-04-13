package models

import "time"

// TagCategory constants define the supported tag categories
const (
	TagCategoryEvent   = "event"   // 时间相关的事件，如发布会、版本更新
	TagCategoryPerson  = "person"  // 具体人物
	TagCategoryKeyword = "keyword" // 关键词，兜底类别（组织、产品、概念等）
)

// TagCategoryMeta defines default display properties for each category
type TagCategoryMeta struct {
	Key         string // category key: event, person, keyword
	Label       string // display label: 事件, 人物, 关键词
	DefaultIcon string // Iconify icon id
	Color       string // default color for nodes/badges
}

// DefaultTagCategories returns the standard category definitions
func DefaultTagCategories() []TagCategoryMeta {
	return []TagCategoryMeta{
		{Key: TagCategoryEvent, Label: "事件", DefaultIcon: "mdi:calendar-star", Color: "#f59e0b"},
		{Key: TagCategoryPerson, Label: "人物", DefaultIcon: "mdi:account", Color: "#10b981"},
		{Key: TagCategoryKeyword, Label: "关键词", DefaultIcon: "mdi:tag", Color: "#6366f1"},
	}
}

// GetCategoryMeta returns the metadata for a category key
func GetCategoryMeta(category string) TagCategoryMeta {
	for _, meta := range DefaultTagCategories() {
		if meta.Key == category {
			return meta
		}
	}
	// Default to keyword
	return TagCategoryMeta{Key: TagCategoryKeyword, Label: "关键词", DefaultIcon: "mdi:tag", Color: "#6366f1"}
}

// TopicTag represents a tag extracted from AI summaries
// Tags are categorized into event, person, or keyword
type TopicTag struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Slug         string    `gorm:"size:120;not null;index:idx_topic_tags_category_slug" json:"slug"`
	Label        string    `gorm:"size:160;not null" json:"label"`
	Category     string    `gorm:"size:20;not null;default:keyword;index:idx_topic_tags_category_slug" json:"category"` // event, person, keyword
	Icon         string    `gorm:"size:100" json:"icon"`                                                                // Iconify icon id, overrides category default
	Aliases      string    `gorm:"type:text" json:"aliases"`                                                            // JSON array of alias strings
	IsCanonical  bool      `gorm:"default:false" json:"is_canonical"`                                                   // true if this is a canonical tag (not merged)
	Source       string    `gorm:"size:20;default:llm" json:"source"`                                                   // llm, heuristic, manual
	FeedCount    int       `gorm:"default:0" json:"feed_count"`                                                         // distinct feed count referencing this tag
	Status       string    `gorm:"size:20;not null;default:active;index" json:"status"`                                 // active, merged
	MergedIntoID *uint     `gorm:"index" json:"merged_into_id,omitempty"`                                               // points to target tag when merged
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	// Kind is retained for backward compatibility, maps to Category
	// Deprecated: Use Category instead
	Kind string `gorm:"size:20;default:keyword" json:"kind"`

	// Relations
	Embedding  *TopicTagEmbedding `gorm:"foreignKey:TopicTagID" json:"embedding,omitempty"`
	MergedInto *TopicTag          `gorm:"foreignKey:MergedIntoID" json:"merged_into,omitempty"`
}

// TableName specifies the table name for TopicTag
func (TopicTag) TableName() string {
	return "topic_tags"
}

// TopicTagEmbedding stores vector embeddings for tag similarity matching
type TopicTagEmbedding struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	TopicTagID   uint      `gorm:"uniqueIndex;not null" json:"topic_tag_id"`
	Vector       string    `gorm:"type:text;not null" json:"vector"` // Deprecated: legacy JSON text payload. Use EmbeddingVec for pgvector.
	EmbeddingVec string    `gorm:"type:vector(1536);column:embedding" json:"-"`
	Dimension    int       `gorm:"not null" json:"dimension"`     // Vector dimension (e.g., 1536 for ada-002)
	Model        string    `gorm:"size:50;not null" json:"model"` // Model used: "text-embedding-ada-002"
	TextHash     string    `gorm:"size:64" json:"text_hash"`      // Hash of (label + aliases + category) for re-embedding detection
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	TopicTag *TopicTag `gorm:"foreignKey:TopicTagID" json:"topic_tag,omitempty"`
}

// TableName specifies the table name for TopicTagEmbedding
func (TopicTagEmbedding) TableName() string {
	return "topic_tag_embeddings"
}

// AISummaryTopic represents the many-to-many relationship between summaries and tags
type AISummaryTopic struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	SummaryID  uint       `gorm:"index;not null" json:"summary_id"`
	TopicTagID uint       `gorm:"index;not null" json:"topic_tag_id"`
	Score      float64    `gorm:"default:0" json:"score"`
	Source     string     `gorm:"size:20;default:llm" json:"source"`
	CreatedAt  time.Time  `json:"created_at"`
	Summary    *AISummary `gorm:"foreignKey:SummaryID;constraint:OnDelete:CASCADE" json:"summary,omitempty"`
	TopicTag   *TopicTag  `gorm:"foreignKey:TopicTagID;constraint:OnDelete:CASCADE" json:"topic_tag,omitempty"`
}

func (AISummaryTopic) TableName() string {
	return "ai_summary_topics"
}

// ArticleTopicTag represents the many-to-many relationship between articles and tags
// This allows individual articles to be tagged for more granular topic tracking
type ArticleTopicTag struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	ArticleID  uint      `gorm:"index:idx_article_topic_tag_article;uniqueIndex:idx_article_topic_tags_link;not null" json:"article_id"`
	TopicTagID uint      `gorm:"index:idx_article_topic_tag_topic;uniqueIndex:idx_article_topic_tags_link;not null" json:"topic_tag_id"`
	Score      float64   `gorm:"default:0" json:"score"`
	Source     string    `gorm:"size:20;default:llm" json:"source"` // llm, heuristic, manual
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`

	// Relations
	Article  *Article  `gorm:"foreignKey:ArticleID;constraint:OnDelete:CASCADE" json:"article,omitempty"`
	TopicTag *TopicTag `gorm:"foreignKey:TopicTagID;constraint:OnDelete:CASCADE" json:"topic_tag,omitempty"`
}

// TableName specifies the table name for ArticleTopicTag
func (ArticleTopicTag) TableName() string {
	return "article_topic_tags"
}
