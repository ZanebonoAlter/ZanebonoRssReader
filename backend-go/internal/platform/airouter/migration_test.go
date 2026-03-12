package airouter

import (
	"testing"

	"github.com/stretchr/testify/require"
	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/platform/aisettings"
)

func TestEnsureLegacySummaryConfigMigratedCreatesDefaultProviderAndRoutes(t *testing.T) {
	db := setupAIRouterTestDB(t)
	require.NoError(t, aisettings.SaveSummaryConfig(map[string]interface{}{
		"base_url":   "https://api.example/v1",
		"api_key":    "token",
		"model":      "gpt-test",
		"time_range": 180,
	}, "legacy summary config"))

	require.NoError(t, EnsureLegacySummaryConfigMigrated())

	var provider models.AIProvider
	require.NoError(t, db.Where("name = ?", DefaultProviderName).First(&provider).Error)
	require.Equal(t, "https://api.example/v1", provider.BaseURL)

	var routeCount int64
	require.NoError(t, db.Model(&models.AIRoute{}).Count(&routeCount).Error)
	require.GreaterOrEqual(t, routeCount, int64(4))

	autoConfig, _, err := aisettings.LoadAutoSummaryConfig()
	require.NoError(t, err)
	require.Equal(t, float64(180), autoConfig["time_range"])
}
