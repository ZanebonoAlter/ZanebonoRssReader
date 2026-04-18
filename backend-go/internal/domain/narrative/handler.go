package narrative

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

var service = NewNarrativeService()

func RegisterNarrativeRoutes(rg *gin.RouterGroup) {
	group := rg.Group("/narratives")
	{
		group.GET("/timeline", getNarrativeTimeline)
		group.GET("", getNarratives)
		group.DELETE("", deleteNarratives)
		group.GET("/:id", getNarrative)
		group.GET("/:id/history", getNarrativeHistory)
	}
}

func getNarrativeTimeline(c *gin.Context) {
	dateStr := c.Query("date")
	var date time.Time
	if dateStr != "" {
		var err error
		date, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid date format, use YYYY-MM-DD"})
			return
		}
	} else {
		date = time.Now()
	}

	daysStr := c.DefaultQuery("days", "7")
	days := 7
	if d, err := strconv.Atoi(daysStr); err == nil && d > 0 {
		days = d
	}

	timeline, err := service.GetTimeline(date, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": timeline})
}

func getNarratives(c *gin.Context) {
	dateStr := c.Query("date")
	var date time.Time
	if dateStr != "" {
		var err error
		date, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid date format, use YYYY-MM-DD"})
			return
		}
	} else {
		date = time.Now()
	}

	narratives, err := service.GetByDate(date)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": narratives})
}

func deleteNarratives(c *gin.Context) {
	dateStr := c.Query("date")
	var date time.Time
	if dateStr != "" {
		var err error
		date, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid date format, use YYYY-MM-DD"})
			return
		}
	} else {
		date = time.Now()
	}

	deleted, err := service.DeleteByDate(date)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"deleted": deleted}})
}

func getNarrative(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid id"})
		return
	}

	narrative, err := service.GetNarrativeTree(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "narrative not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": narrative})
}

func getNarrativeHistory(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid id"})
		return
	}

	history, err := service.GetNarrativeHistory(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": history})
}
