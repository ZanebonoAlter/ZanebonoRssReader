package topicgraph

import (
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
