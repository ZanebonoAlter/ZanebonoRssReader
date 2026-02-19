package services

import (
	"my-robot-backend/internal/models"
	"my-robot-backend/pkg/database"
)

func GetFirecrawlConfig() (*FirecrawlConfig, error) {
	var settings models.AISettings
	if err := database.DB.Where("key = ?", "summary_config").First(&settings).Error; err != nil {
		return nil, err
	}

	settingsDict := settings.ToDict()
	if settingsDict == nil {
		return nil, ErrInvalidSettingsFormat
	}

	settingsData, ok := settingsDict["value"].(map[string]interface{})
	if !ok {
		return nil, ErrInvalidSettingsFormat
	}

	firecrawlData, ok := settingsData["firecrawl"].(map[string]interface{})
	if !ok {
		return nil, ErrFirecrawlConfigMissing
	}

	config := &FirecrawlConfig{
		APIUrl:           getStringValue(firecrawlData, "api_url"),
		APIKey:           getStringValue(firecrawlData, "api_key"),
		Enabled:          getBoolValue(firecrawlData, "enabled"),
		Mode:             getStringValue(firecrawlData, "mode"),
		Timeout:          getIntValue(firecrawlData, "timeout"),
		MaxContentLength: getIntValue(firecrawlData, "max_content_length"),
	}

	if config.Timeout <= 0 {
		config.Timeout = 60
	}
	if config.MaxContentLength <= 0 {
		config.MaxContentLength = 50000
	}

	return config, nil
}

func getStringValue(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return ""
}

func getBoolValue(m map[string]interface{}, key string) bool {
	if val, ok := m[key]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return false
}

func getIntValue(m map[string]interface{}, key string) int {
	if val, ok := m[key]; ok {
		switch v := val.(type) {
		case float64:
			return int(v)
		case int:
			return v
		}
	}
	return 0
}

var (
	ErrInvalidSettingsFormat  = &FirecrawlError{Message: "invalid settings format"}
	ErrFirecrawlConfigMissing = &FirecrawlError{Message: "firecrawl configuration missing"}
)

type FirecrawlError struct {
	Message string
}

func (e *FirecrawlError) Error() string {
	return e.Message
}
