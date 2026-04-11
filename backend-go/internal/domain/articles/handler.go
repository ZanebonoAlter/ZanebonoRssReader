package articles

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/domain/topicextraction"
	"my-robot-backend/internal/platform/database"
)

func loadArticleWithTagCount(articleID uint) (*models.Article, error) {
	var article models.Article
	if err := database.DB.Model(&models.Article{}).
		Joins("LEFT JOIN feeds ON articles.feed_id = feeds.id").
		Joins("LEFT JOIN (SELECT article_id, COUNT(*) AS tag_count FROM article_topic_tags GROUP BY article_id) tag_stats ON tag_stats.article_id = articles.id").
		Select("articles.*, feeds.category_id AS category_id, COALESCE(tag_stats.tag_count, 0) AS tag_count").
		First(&article, articleID).Error; err != nil {
		return nil, err
	}

	return &article, nil
}

type UpdateArticleRequest struct {
	Read     *bool `json:"read"`
	Favorite *bool `json:"favorite"`
}

type BulkUpdateArticlesRequest struct {
	IDs           []uint `json:"ids"`
	FeedID        *uint  `json:"feed_id"`
	CategoryID    *uint  `json:"category_id"`
	Uncategorized *bool  `json:"uncategorized"`
	Read          *bool  `json:"read"`
	Favorite      *bool  `json:"favorite"`
}

func GetArticlesStats(c *gin.Context) {
	var total, unread, favorite int64

	database.DB.Model(&models.Article{}).Count(&total)
	database.DB.Model(&models.Article{}).Where("read = ?", false).Count(&unread)
	database.DB.Model(&models.Article{}).Where("favorite = ?", true).Count(&favorite)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"total":    total,
			"unread":   unread,
			"favorite": favorite,
		},
	})
}

func GetArticles(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	feedID, _ := strconv.Atoi(c.Query("feed_id"))
	categoryID, _ := strconv.Atoi(c.Query("category_id"))
	uncategorized := c.Query("uncategorized") == "true"
	read := c.Query("read")
	favorite := c.Query("favorite")
	search := c.Query("search")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	maxPerPage := 100
	if perPage <= 0 || perPage > maxPerPage {
		perPage = maxPerPage
	}

	query := database.DB.Model(&models.Article{}).
		Joins("LEFT JOIN feeds ON articles.feed_id = feeds.id").
		Joins("LEFT JOIN (SELECT article_id, COUNT(*) AS tag_count FROM article_topic_tags GROUP BY article_id) tag_stats ON tag_stats.article_id = articles.id").
		Select("articles.*, feeds.category_id AS category_id, COALESCE(tag_stats.tag_count, 0) AS tag_count")

	if feedID > 0 {
		query = query.Where("articles.feed_id = ?", feedID)
	}

	if categoryID > 0 {
		query = query.Where("feeds.category_id = ?", categoryID)
	}

	if uncategorized {
		query = query.Where("feeds.category_id IS NULL")
	}

	if read == "true" || read == "false" {
		query = query.Where("articles.read = ?", read == "true")
	}

	if favorite == "true" || favorite == "false" {
		query = query.Where("articles.favorite = ?", favorite == "true")
	}

	if search != "" {
		searchTerm := "%" + search + "%"
		query = query.Where("articles.title LIKE ? OR articles.description LIKE ?", searchTerm, searchTerm)
	}

	if startDate != "" {
		query = query.Where("DATE(articles.pub_date) >= ?", startDate)
	}

	if endDate != "" {
		query = query.Where("DATE(articles.pub_date) <= ?", endDate)
	}

	query = query.Order("articles.pub_date DESC")

	var total int64
	query.Count(&total)

	var articles []models.Article
	offset := (page - 1) * perPage
	if err := query.Offset(offset).Limit(perPage).Find(&articles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	data := make([]map[string]interface{}, len(articles))
	for i, article := range articles {
		data[i] = article.ToDict()
	}

	pages := int(total) / perPage
	if int(total)%perPage > 0 {
		pages++
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
		"pagination": gin.H{
			"page":     page,
			"per_page": perPage,
			"total":    total,
			"pages":    pages,
		},
	})
}

func GetArticle(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("article_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid article ID",
		})
		return
	}

	article, err := loadArticleWithTagCount(uint(id))
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "Article not found",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   err.Error(),
			})
		}
		return
	}

	articleData := article.ToDict()
	tags, err := topicextraction.GetArticleTags(article.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}
	articleData["tags"] = tags

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    articleData,
	})
}

func RetagArticleHandler(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("article_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid article ID"})
		return
	}

	var article models.Article
	if err := database.DB.First(&article, uint(id)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Article not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	var feed models.Feed
	if err := database.DB.First(&feed, article.FeedID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	queue := topicextraction.NewTagJobQueue(database.DB)
	if err := queue.Enqueue(topicextraction.TagJobRequest{
		ArticleID:  article.ID,
		FeedName:   feed.Title,
		ForceRetag: true,
		Reason:     "manual_api_trigger",
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	// Query both pending and leased: Enqueue may reuse an existing leased job,
	// or a worker may claim the job between Enqueue and this query.
	var tagJob models.TagJob
	if err := database.DB.Where("article_id = ? AND status IN ?", article.ID,
		[]string{string(models.JobStatusPending), string(models.JobStatusLeased)}).
		Order("id DESC").
		First(&tagJob).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "failed to retrieve job_id"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "标签任务已提交，请稍后刷新查看结果",
		"data": gin.H{
			"job_id":     tagJob.ID,
			"article_id": article.ID,
			"status":     tagJob.Status,
		},
	})
}

func UpdateArticle(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("article_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid article ID",
		})
		return
	}

	var article models.Article
	if err := database.DB.First(&article, uint(id)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "Article not found",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   err.Error(),
			})
		}
		return
	}

	var req UpdateArticleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request body",
		})
		return
	}

	updates := make(map[string]interface{})
	if req.Read != nil {
		updates["read"] = *req.Read
	}
	if req.Favorite != nil {
		updates["favorite"] = *req.Favorite
	}

	if len(updates) > 0 {
		if err := database.DB.Model(&article).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   err.Error(),
			})
			return
		}
	}

	database.DB.Model(&models.Article{}).
		Joins("LEFT JOIN feeds ON articles.feed_id = feeds.id").
		Joins("LEFT JOIN (SELECT article_id, COUNT(*) AS tag_count FROM article_topic_tags GROUP BY article_id) tag_stats ON tag_stats.article_id = articles.id").
		Select("articles.*, feeds.category_id AS category_id, COALESCE(tag_stats.tag_count, 0) AS tag_count").
		First(&article, uint(id))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    article.ToDict(),
	})
}

func BulkUpdateArticles(c *gin.Context) {
	var req BulkUpdateArticlesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	updates := make(map[string]interface{})
	if req.Read != nil {
		updates["read"] = *req.Read
	}
	if req.Favorite != nil {
		updates["favorite"] = *req.Favorite
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "At least one field (read or favorite) must be specified",
		})
		return
	}

	query := database.DB.Model(&models.Article{})

	if len(req.IDs) > 0 {
		query = query.Where("id IN ?", req.IDs)
	} else if req.FeedID != nil {
		query = query.Where("feed_id = ?", *req.FeedID)
	} else if req.CategoryID != nil {
		query = query.Where("feed_id IN (SELECT id FROM feeds WHERE category_id = ?)", *req.CategoryID)
	} else if req.Uncategorized != nil && *req.Uncategorized {
		query = query.Where("feed_id IN (SELECT id FROM feeds WHERE category_id IS NULL)")
	}

	result := query.Updates(updates)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   result.Error.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": result.RowsAffected,
	})
}
