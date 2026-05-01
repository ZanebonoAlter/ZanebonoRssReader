package narrative

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/platform/logging"
)

func registerConceptRoutes(rg *gin.RouterGroup) {
	conceptGroup := rg.Group("/narratives/board-concepts")
	{
		conceptGroup.GET("", listBoardConcepts)
		conceptGroup.POST("", createBoardConcept)
		conceptGroup.PUT("/:id", updateBoardConcept)
		conceptGroup.DELETE("/:id", deactivateBoardConcept)
		conceptGroup.POST("/suggest", suggestConcepts)
	}
	rg.GET("/narratives/unclassified", getUnclassifiedTags)
}

func getUnclassifiedTags(c *gin.Context) {
	tags := GetUnclassifiedBucket()
	if tags == nil {
		tags = []TagInput{}
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": tags})
}

func listBoardConcepts(c *gin.Context) {
	concepts, err := ListActiveConcepts()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	if concepts == nil {
		concepts = []models.BoardConcept{}
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": concepts})
}

type createConceptRequest struct {
	Name            string `json:"name"`
	Description     string `json:"description"`
	ScopeType       string `json:"scope_type"`
	ScopeCategoryID *uint  `json:"scope_category_id"`
}

func createBoardConcept(c *gin.Context) {
	var req createConceptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid request body"})
		return
	}

	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "name is required"})
		return
	}

	if req.ScopeType == "" {
		req.ScopeType = "global"
	}

	concept, err := CreateConcept(req.Name, req.Description, req.ScopeType, req.ScopeCategoryID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if embErr := GenerateConceptEmbedding(ctx, concept); embErr != nil {
		logging.Warnf("concept: failed to generate embedding for concept %d: %v", concept.ID, embErr)
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": concept})
}

type updateConceptRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func updateBoardConcept(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid id"})
		return
	}

	var req updateConceptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid request body"})
		return
	}

	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "name is required"})
		return
	}

	concept, err := UpdateConcept(uint(id), req.Name, req.Description)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if embErr := GenerateConceptEmbedding(ctx, concept); embErr != nil {
		logging.Warnf("concept: failed to regenerate embedding for concept %d: %v", concept.ID, embErr)
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": concept})
}

func deactivateBoardConcept(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid id"})
		return
	}

	if err := DeactivateConcept(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"deactivated": true}})
}

func suggestConcepts(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	suggestions, err := SuggestBoardConcepts(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	if suggestions == nil {
		suggestions = []ConceptSuggestion{}
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": suggestions})
}
