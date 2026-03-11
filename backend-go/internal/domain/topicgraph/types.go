package topicgraph

type ExtractionInput struct {
	Title        string
	Summary      string
	FeedName     string
	CategoryName string
}

type TopicTag struct {
	Label string  `json:"label"`
	Slug  string  `json:"slug"`
	Kind  string  `json:"kind"`
	Score float64 `json:"score"`
}

type GraphNode struct {
	ID           string  `json:"id"`
	Label        string  `json:"label"`
	Slug         string  `json:"slug,omitempty"`
	Kind         string  `json:"kind"`
	Weight       float64 `json:"weight"`
	SummaryCount int     `json:"summary_count,omitempty"`
	Color        string  `json:"color,omitempty"`
	FeedName     string  `json:"feed_name,omitempty"`
	CategoryName string  `json:"category_name,omitempty"`
}

type GraphEdge struct {
	ID     string  `json:"id"`
	Source string  `json:"source"`
	Target string  `json:"target"`
	Kind   string  `json:"kind"`
	Weight float64 `json:"weight"`
}

type TopicSummaryCard struct {
	ID           uint               `json:"id"`
	Title        string             `json:"title"`
	Summary      string             `json:"summary"`
	FeedName     string             `json:"feed_name"`
	FeedColor    string             `json:"feed_color"`
	CategoryName string             `json:"category_name"`
	ArticleCount int                `json:"article_count"`
	CreatedAt    string             `json:"created_at"`
	Topics       []TopicTag         `json:"topics"`
	Articles     []TopicArticleCard `json:"articles"`
}

type TopicArticleCard struct {
	ID    uint   `json:"id"`
	Title string `json:"title"`
	Link  string `json:"link"`
}

type TopicHistoryPoint struct {
	AnchorDate string `json:"anchor_date"`
	Count      int    `json:"count"`
	Label      string `json:"label"`
}

type TopicDetail struct {
	Topic         TopicTag            `json:"topic"`
	Summaries     []TopicSummaryCard  `json:"summaries"`
	History       []TopicHistoryPoint `json:"history"`
	RelatedTopics []TopicTag          `json:"related_topics"`
	SearchLinks   map[string]string   `json:"search_links"`
	AppLinks      map[string]string   `json:"app_links"`
}

type TopicGraphResponse struct {
	Type         string      `json:"type"`
	AnchorDate   string      `json:"anchor_date"`
	PeriodLabel  string      `json:"period_label"`
	Nodes        []GraphNode `json:"nodes"`
	Edges        []GraphEdge `json:"edges"`
	TopicCount   int         `json:"topic_count"`
	SummaryCount int         `json:"summary_count"`
	FeedCount    int         `json:"feed_count"`
	TopTopics    []TopicTag  `json:"top_topics"`
}
