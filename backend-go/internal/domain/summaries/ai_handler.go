package summaries

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"my-robot-backend/internal/domain/contentprocessing"
	"my-robot-backend/internal/domain/models"
	platformai "my-robot-backend/internal/platform/ai"
	"my-robot-backend/internal/platform/airouter"
	"my-robot-backend/internal/platform/aisettings"
	"my-robot-backend/internal/platform/database"
)

type SummarizeArticleRequest struct {
	BaseURL  string `json:"base_url"`
	APIKey   string `json:"api_key"`
	Model    string `json:"model"`
	Title    string `json:"title" binding:"required"`
	Content  string `json:"content" binding:"required"`
	Language string `json:"language"`
}

type TestAIConnectionRequest struct {
	BaseURL      string `json:"base_url" binding:"required"`
	APIKey       string `json:"api_key"`
	Model        string `json:"model" binding:"required"`
	ProviderType string `json:"provider_type"`
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
			"error":   "API 密钥是必填项",
		})
		return
	}

	language := req.Language
	if language == "" {
		language = "zh"
	}

	var summary *platformai.AISummaryResponse
	var err error
	if strings.TrimSpace(req.BaseURL) != "" && strings.TrimSpace(req.APIKey) != "" && strings.TrimSpace(req.Model) != "" {
		aiService := platformai.NewAIService(req.BaseURL, req.APIKey, req.Model)
		summary, err = aiService.SummarizeArticle(req.Title, req.Content, language)
	} else {
		router := airouter.NewRouter()
		maxTokens := 16000
		result, routeErr := router.Chat(context.Background(), airouter.ChatRequest{
			Capability: airouter.CapabilitySummary,
			Messages: []airouter.Message{
				{Role: "system", Content: buildArticleSummarySystemPrompt(language)},
				{Role: "user", Content: buildArticleSummaryUserPrompt(req.Title, req.Content)},
			},
			MaxTokens: &maxTokens,
			Metadata: map[string]any{
				"title":    req.Title,
				"language": language,
			},
		})
		if routeErr != nil {
			err = routeErr
		} else {
			summary = platformai.ParseSummaryMarkdown(result.Content)
		}
	}
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
			"error":   "缺少必填字段",
		})
		return
	}

	if req.ProviderType != "ollama" && strings.TrimSpace(req.APIKey) == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "API Key 是必填项",
		})
		return
	}

	aiService := platformai.NewAIService(req.BaseURL, req.APIKey, req.Model)

	if err := aiService.TestConnection(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "连接测试失败：" + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "连接测试成功",
	})
}

func GetAISettings(c *gin.Context) {
	router := airouter.NewRouter()
	provider, route, err := router.ResolvePrimaryProvider(airouter.CapabilitySummary)
	if err == nil && provider != nil && route != nil {
		autoSummaryConfig, _, _ := aisettings.LoadAutoSummaryConfig()
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"base_url":           provider.BaseURL,
				"model":              provider.Model,
				"provider_id":        provider.ID,
				"provider_name":      provider.Name,
				"route_name":         route.Name,
				"time_range":         autoSummaryConfig["time_range"],
				"api_key_configured": strings.TrimSpace(provider.APIKey) != "",
			},
		})
		return
	}

	var settings models.AISettings
	if legacyErr := database.DB.Where("key = ?", "summary_config").First(&settings).Error; legacyErr != nil {
		c.JSON(http.StatusOK, gin.H{"success": true, "data": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": settings.ToDict()})
}

func SaveAISettings(c *gin.Context) {
	var req SaveAISettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "缺少必填字段",
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

	configJSON, _, err := aisettings.LoadSummaryConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	configJSON["base_url"] = baseURL
	configJSON["api_key"] = req.APIKey
	configJSON["model"] = model

	if err := aisettings.SaveSummaryConfig(configJSON, "AI summary generation configuration"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	store := airouter.NewStore(database.DB)
	if _, err := store.EnsureLegacyProviderAndRoutes(baseURL, req.APIKey, model); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	contentprocessing.SetCompletionAICredentials(baseURL, req.APIKey, model)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "AI 设置保存成功",
	})
}

func buildArticleSummarySystemPrompt(language string) string {
	service := platformai.NewAIService("", "", "")
	return service.GetSystemPrompt(language)
}

func buildArticleSummaryUserPrompt(title, content string) string {
	service := platformai.NewAIService("", "", "")
	return service.PrepareArticleContent(title, content)
}
