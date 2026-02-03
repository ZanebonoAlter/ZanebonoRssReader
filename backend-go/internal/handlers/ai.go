package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"my-robot-backend/internal/models"
	"my-robot-backend/internal/services"
	"my-robot-backend/pkg/database"
)

type SummarizeArticleRequest struct {
	BaseURL  string `json:"base_url" binding:"required"`
	APIKey   string `json:"api_key" binding:"required"`
	Model    string `json:"model" binding:"required"`
	Title    string `json:"title" binding:"required"`
	Content  string `json:"content" binding:"required"`
	Language string `json:"language"`
}

type TestAIConnectionRequest struct {
	BaseURL string `json:"base_url" binding:"required"`
	APIKey  string `json:"api_key" binding:"required"`
	Model   string `json:"model" binding:"required"`
}

type SaveAISettingsRequest struct {
	BaseURL string `json:"base_url"`
	APIKey  string `json:"api_key" binding:"required"`
	Model   string `json:"model"`
}

func SummarizeArticle(c *gin.Context) {
	var req SummarizeArticleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Missing required fields",
		})
		return
	}

	aiService := services.NewAIService(req.BaseURL, req.APIKey, req.Model)

	language := req.Language
	if language == "" {
		language = "zh"
	}

	summary, err := aiService.SummarizeArticle(req.Title, req.Content, language)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    summary,
	})
}

func TestAIConnection(c *gin.Context) {
	var req TestAIConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Missing required fields",
		})
		return
	}

	aiService := services.NewAIService(req.BaseURL, req.APIKey, req.Model)

	if err := aiService.TestConnection(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "连接测试失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "连接测试成功",
	})
}

func GetAISettings(c *gin.Context) {
	var settings models.AISettings
	if err := database.DB.Where("key = ?", "summary_config").First(&settings).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    settings.ToDict(),
	})
}

func SaveAISettings(c *gin.Context) {
	var req SaveAISettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "API key is required",
		})
		return
	}

	baseURL := req.BaseURL
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	model := req.Model
	if model == "" {
		model = "gpt-4o-mini"
	}

	configJSON := map[string]interface{}{
		"base_url": baseURL,
		"api_key":  req.APIKey,
		"model":    model,
	}

	configBytes, err := json.Marshal(configJSON)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	var settings models.AISettings
	err = database.DB.Where("key = ?", "summary_config").First(&settings).Error

	if err == nil {
		settings.Value = string(configBytes)
		database.DB.Save(&settings)
	} else {
		settings = models.AISettings{
			Key:         "summary_config",
			Value:       string(configBytes),
			Description: "AI summary generation configuration",
		}
		database.DB.Create(&settings)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "AI settings saved successfully",
	})
}
