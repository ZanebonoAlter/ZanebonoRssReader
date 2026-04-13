package topicanalysis

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/domain/topictypes"
	"my-robot-backend/internal/platform/database"
)

// mergePreviewResponse wraps a single candidate with optional article lists.
type mergePreviewCandidate struct {
	TagMergeCandidate
	SourceArticleList []CandidateArticle `json:"source_article_list,omitempty"`
	TargetArticleList []CandidateArticle `json:"target_article_list,omitempty"`
}

// ScanMergePreviewHandler returns candidate tag pairs for manual review.
// GET /api/topic-tags/merge-preview?limit=50&include_articles=false
func ScanMergePreviewHandler(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "50")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	includeArticles := c.DefaultQuery("include_articles", "false") == "true"

	candidates, err := ScanSimilarTagPairs(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	result := make([]mergePreviewCandidate, 0, len(candidates))
	for _, cand := range candidates {
		item := mergePreviewCandidate{TagMergeCandidate: cand}
		if includeArticles {
			if arts, err := GetCandidateArticleTitles(cand.SourceTagID, 5); err == nil {
				item.SourceArticleList = arts
			}
			if arts, err := GetCandidateArticleTitles(cand.TargetTagID, 5); err == nil {
				item.TargetArticleList = arts
			}
		}
		result = append(result, item)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"candidates": result,
			"total":      len(result),
		},
	})
}

// MergeTagsWithCustomNameHandler merges two tags and optionally renames the target.
// POST /api/topic-tags/merge-with-name
func MergeTagsWithCustomNameHandler(c *gin.Context) {
	var body struct {
		SourceTagID uint   `json:"source_tag_id"`
		TargetTagID uint   `json:"target_tag_id"`
		NewName     string `json:"new_name"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid request body"})
		return
	}

	if body.SourceTagID == 0 || body.TargetTagID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "source_tag_id and target_tag_id are required"})
		return
	}

	if body.SourceTagID == body.TargetTagID {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "cannot merge tag into itself"})
		return
	}

	newName := strings.TrimSpace(body.NewName)
	if newName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "new_name is required"})
		return
	}

	var source, target models.TopicTag
	if err := database.DB.First(&source, body.SourceTagID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "source tag not found"})
		return
	}
	if err := database.DB.First(&target, body.TargetTagID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "target tag not found"})
		return
	}

	if source.Status == "merged" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "source tag is already merged"})
		return
	}
	if target.Status == "merged" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "target tag is already merged"})
		return
	}

	// Rename target if new_name differs from current label
	if newName != target.Label {
		newSlug := topictypes.Slugify(newName)
		if err := database.DB.Model(&target).Updates(map[string]interface{}{
			"label": newName,
			"slug":  newSlug,
		}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "failed to rename target tag"})
			return
		}
		target.Label = newName
		target.Slug = newSlug
	}

	if err := MergeTags(body.SourceTagID, body.TargetTagID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"source_id": body.SourceTagID,
			"target_id": body.TargetTagID,
			"new_label": target.Label,
			"merged_at": time.Now().Format(time.RFC3339),
		},
	})
}

// RegisterTagMergePreviewRoutes registers the preview and custom-name merge endpoints.
func RegisterTagMergePreviewRoutes(rg *gin.RouterGroup) {
	tags := rg.Group("/topic-tags")
	{
		tags.GET("/merge-preview", ScanMergePreviewHandler)
		tags.POST("/merge-with-name", MergeTagsWithCustomNameHandler)
	}
}
