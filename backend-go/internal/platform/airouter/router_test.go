package airouter

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"my-robot-backend/internal/domain/models"
)

type fakeProviderClient struct {
	responses map[string]struct {
		content string
		err     error
	}
}

func (f *fakeProviderClient) Chat(_ context.Context, provider models.AIProvider, _ ChatRequest) (string, error) {
	res := f.responses[provider.Name]
	return res.content, res.err
}

func TestRouterFallsBackOnRetryableProviderError(t *testing.T) {
	db := setupAIRouterTestDB(t)
	store := NewStore(db)

	p1 := models.AIProvider{Name: "primary", ProviderType: ProviderTypeOpenAICompatible, BaseURL: "https://a.example/v1", APIKey: "a", Model: "m1", Enabled: true}
	p2 := models.AIProvider{Name: "backup", ProviderType: ProviderTypeOpenAICompatible, BaseURL: "https://b.example/v1", APIKey: "b", Model: "m2", Enabled: true}
	require.NoError(t, db.Create(&p1).Error)
	require.NoError(t, db.Create(&p2).Error)
	route := models.AIRoute{Name: DefaultRouteName, Capability: string(CapabilitySummary), Enabled: true, Strategy: "ordered_failover"}
	require.NoError(t, db.Create(&route).Error)
	require.NoError(t, db.Create(&models.AIRouteProvider{RouteID: route.ID, ProviderID: p1.ID, Priority: 1, Enabled: true}).Error)
	require.NoError(t, db.Create(&models.AIRouteProvider{RouteID: route.ID, ProviderID: p2.ID, Priority: 2, Enabled: true}).Error)

	router := NewRouterWithStore(store)
	router.RegisterClient(ProviderTypeOpenAICompatible, &fakeProviderClient{responses: map[string]struct {
		content string
		err     error
	}{
		"primary": {err: &ProviderError{Message: "rate limited", Code: "rate_limit", Retryable: true}},
		"backup":  {content: "ok from backup"},
	}})

	result, err := router.Chat(context.Background(), ChatRequest{Capability: CapabilitySummary, Messages: []Message{{Role: "user", Content: "hi"}}})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "ok from backup", result.Content)
	require.True(t, result.UsedFallback)
	require.Equal(t, 2, result.AttemptCount)
}

func TestRouterStopsOnTerminalError(t *testing.T) {
	db := setupAIRouterTestDB(t)
	store := NewStore(db)

	p1 := models.AIProvider{Name: "primary-terminal", ProviderType: ProviderTypeOpenAICompatible, BaseURL: "https://a.example/v1", APIKey: "a", Model: "m1", Enabled: true}
	p2 := models.AIProvider{Name: "backup-never-used", ProviderType: ProviderTypeOpenAICompatible, BaseURL: "https://b.example/v1", APIKey: "b", Model: "m2", Enabled: true}
	require.NoError(t, db.Create(&p1).Error)
	require.NoError(t, db.Create(&p2).Error)
	route := models.AIRoute{Name: DefaultRouteName, Capability: string(CapabilitySummary), Enabled: true, Strategy: "ordered_failover"}
	require.NoError(t, db.Create(&route).Error)
	require.NoError(t, db.Create(&models.AIRouteProvider{RouteID: route.ID, ProviderID: p1.ID, Priority: 1, Enabled: true}).Error)
	require.NoError(t, db.Create(&models.AIRouteProvider{RouteID: route.ID, ProviderID: p2.ID, Priority: 2, Enabled: true}).Error)

	router := NewRouterWithStore(store)
	router.RegisterClient(ProviderTypeOpenAICompatible, &fakeProviderClient{responses: map[string]struct {
		content string
		err     error
	}{
		"primary-terminal":  {err: &ProviderError{Message: "invalid key", Code: "unauthorized", Retryable: false}},
		"backup-never-used": {content: "should not be used"},
	}})

	result, err := router.Chat(context.Background(), ChatRequest{Capability: CapabilitySummary, Messages: []Message{{Role: "user", Content: "hi"}}})
	require.Error(t, err)
	require.Nil(t, result)
}
