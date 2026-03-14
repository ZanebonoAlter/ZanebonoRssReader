package topicgraph

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func GetTopicGraph(c *gin.Context) {
	kind := c.Param("type")
	anchor, err := parseAnchorDate(c.Query("date"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	graph, err := BuildTopicGraph(kind, anchor)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": graph})
}

func GetTopicDetail(c *gin.Context) {
	slug := c.Param("slug")
	kind := c.DefaultQuery("type", "daily")
	anchor, err := parseAnchorDate(c.Query("date"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	detail, err := BuildTopicDetail(kind, slug, anchor)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": detail})
}

func parseAnchorDate(value string) (time.Time, error) {
	if value == "" {
		return time.Now().In(topicGraphCST), nil
	}
	parsed, err := time.ParseInLocation("2006-01-02", value, topicGraphCST)
	if err != nil {
		return time.Time{}, err
	}
	return parsed, nil
}

// GetTopicsByCategory returns tags grouped by category (event, person, keyword)
func GetTopicsByCategory(c *gin.Context) {
	kind := c.DefaultQuery("type", "daily")
	anchor, err := parseAnchorDate(c.Query("date"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	result, err := BuildTopicsByCategory(kind, anchor)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}

// GetTopicArticles returns paginated articles for a topic
func GetTopicArticles(c *gin.Context) {
	slug := c.Param("slug")
	if slug == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "slug is required"})
		return
	}

	// Parse query parameters
	kind := c.DefaultQuery("type", "daily")
	page := 1
	pageSize := 15

	if p := c.Query("page"); p != "" {
		if parsed, err := parseIntParam(p, 1, 1000); err == nil {
			page = parsed
		}
	}
	if ps := c.Query("page_size"); ps != "" {
		if parsed, err := parseIntParam(ps, 1, 100); err == nil {
			pageSize = parsed
		}
	}

	anchor, err := parseAnchorDate(c.Query("date"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	articles, total, err := FetchTopicArticles(slug, kind, anchor, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"articles":  articles,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// parseIntParam parses an integer query parameter with bounds checking
func parseIntParam(value string, min, max int) (int, error) {
	var result int
	if _, err := fmt.Sscanf(value, "%d", &result); err != nil {
		return 0, err
	}
	if result < min {
		return min, nil
	}
	if result > max {
		return max, nil
	}
	return result, nil
}
