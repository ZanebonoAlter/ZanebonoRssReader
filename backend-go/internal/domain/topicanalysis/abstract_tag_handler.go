package topicanalysis

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// GetTagHierarchyHandler returns the tag hierarchy tree.
// GET /api/topic-tags/hierarchy?category=
func GetTagHierarchyHandler(c *gin.Context) {
	category := strings.TrimSpace(c.Query("category"))

	nodes, err := GetTagHierarchy(category)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"nodes": nodes,
			"total": len(nodes),
		},
	})
}

// UpdateAbstractTagNameHandler renames an abstract tag.
// PUT /api/topic-tags/:id/abstract-name
func UpdateAbstractTagNameHandler(c *gin.Context) {
	tagIDStr := c.Param("id")
	tagID, err := strconv.ParseUint(tagIDStr, 10, 32)
	if err != nil || tagID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid tag id"})
		return
	}

	var body struct {
		NewName string `json:"new_name"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid request body"})
		return
	}

	newName := strings.TrimSpace(body.NewName)
	if newName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "new_name is required"})
		return
	}
	if len(newName) > 160 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "new_name exceeds 160 characters"})
		return
	}

	if err := UpdateAbstractTagName(uint(tagID), newName); err != nil {
		status := http.StatusInternalServerError
		errMsg := err.Error()
		if strings.Contains(errMsg, "not an abstract tag") || strings.Contains(errMsg, "must be") {
			status = http.StatusBadRequest
		} else if strings.Contains(errMsg, "already in use") {
			status = http.StatusConflict
		}
		c.JSON(status, gin.H{"success": false, "error": errMsg})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"id":       tagID,
			"new_name": newName,
		},
	})
}

// DetachChildTagHandler removes a child tag from its abstract parent.
// POST /api/topic-tags/:id/detach
func DetachChildTagHandler(c *gin.Context) {
	parentIDStr := c.Param("id")
	parentID, err := strconv.ParseUint(parentIDStr, 10, 32)
	if err != nil || parentID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid parent tag id"})
		return
	}

	var body struct {
		ChildID uint `json:"child_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid request body"})
		return
	}
	if body.ChildID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "child_id is required"})
		return
	}

	if err := DetachChildTag(uint(parentID), body.ChildID); err != nil {
		status := http.StatusInternalServerError
		errMsg := err.Error()
		if strings.Contains(errMsg, "must be") || strings.Contains(errMsg, "no relation found") {
			status = http.StatusBadRequest
		}
		c.JSON(status, gin.H{"success": false, "error": errMsg})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "child detached",
	})
}

// RegisterAbstractTagRoutes registers the abstract tag management endpoints.
func RegisterAbstractTagRoutes(rg *gin.RouterGroup) {
	tags := rg.Group("/topic-tags")
	{
		tags.GET("/hierarchy", GetTagHierarchyHandler)
		tags.PUT("/:id/abstract-name", UpdateAbstractTagNameHandler)
		tags.POST("/:id/detach", DetachChildTagHandler)
	}
}
