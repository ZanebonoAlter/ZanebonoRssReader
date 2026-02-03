package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"my-robot-backend/internal/models"
	"my-robot-backend/pkg/database"
	"gorm.io/gorm"
)

type UpdateArticleRequest struct {
	Read     *bool `json:"read"`
	Favorite *bool `json:"favorite"`
}

type BulkUpdateArticlesRequest struct {
	IDs      []uint `json:"ids" binding:"required"`
	Read     *bool  `json:"read"`
	Favorite *bool  `json:"favorite"`
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

	query := database.DB.Model(&models.Article{})

	if feedID > 0 {
		query = query.Where("feed_id = ?", feedID)
	}

	if categoryID > 0 {
		query = query.Joins("JOIN feeds ON articles.feed_id = feeds.id").
			Where("feeds.category_id = ?", categoryID)
	}

	if uncategorized {
		query = query.Joins("JOIN feeds ON articles.feed_id = feeds.id").
			Where("feeds.category_id IS NULL")
	}

	if read == "true" || read == "false" {
		query = query.Where("read = ?", read == "true")
	}

	if favorite == "true" || favorite == "false" {
		query = query.Where("favorite = ?", favorite == "true")
	}

	if search != "" {
		searchTerm := "%" + search + "%"
		query = query.Where("title LIKE ? OR description LIKE ?", searchTerm, searchTerm)
	}

	query = query.Order("pub_date DESC")

	var total int64
	query.Count(&total)

	var articles []models.Article
	if perPage >= 10000 {
		if err := query.Find(&articles).Error; err != nil {
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

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    data,
			"pagination": gin.H{
				"page":     1,
				"per_page": len(articles),
				"total":    total,
				"pages":    1,
			},
		})
		return
	}

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

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    article.ToDict(),
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

	database.DB.First(&article, uint(id))

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
			"error":   "ids is required",
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

	result := database.DB.Model(&models.Article{}).
		Where("id IN ?", req.IDs).
		Updates(updates)

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
