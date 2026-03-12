package airouter

import (
	"context"
	"errors"
	"fmt"
	"time"

	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/platform/database"
)

type Router struct {
	store   *Store
	clients map[string]ProviderClient
}

func NewRouter() *Router {
	store := NewStore(database.DB)
	return &Router{
		store: store,
		clients: map[string]ProviderClient{
			ProviderTypeOpenAICompatible: NewOpenAICompatibleClient(),
		},
	}
}

func NewRouterWithStore(store *Store) *Router {
	return &Router{
		store: store,
		clients: map[string]ProviderClient{
			ProviderTypeOpenAICompatible: NewOpenAICompatibleClient(),
		},
	}
}

func (r *Router) RegisterClient(providerType string, client ProviderClient) {
	if client == nil {
		return
	}
	r.clients[providerType] = client
}

func (r *Router) Chat(ctx context.Context, req ChatRequest) (*ChatResult, error) {
	route, providers, err := r.store.LoadRouteWithProviders(req.Capability)
	if err != nil {
		return nil, err
	}

	var attemptErrors []error
	for idx, provider := range providers {
		client := r.clients[provider.ProviderType]
		if client == nil {
			attemptErrors = append(attemptErrors, fmt.Errorf("provider type %s unsupported", provider.ProviderType))
			continue
		}

		start := time.Now()
		content, callErr := client.Chat(ctx, provider, req)
		latencyMs := int(time.Since(start).Milliseconds())
		if callErr == nil {
			r.store.LogCall(&models.AICallLog{
				Capability:   string(req.Capability),
				RouteName:    route.Name,
				ProviderName: provider.Name,
				Success:      true,
				IsFallback:   idx > 0,
				LatencyMs:    latencyMs,
				RequestMeta:  encodeMeta(req.Metadata),
			})
			return &ChatResult{Content: content, ProviderID: provider.ID, ProviderName: provider.Name, RouteName: route.Name, UsedFallback: idx > 0, AttemptCount: idx + 1}, nil
		}

		providerErr := &ProviderError{}
		retryable := false
		code := "provider_error"
		if errors.As(callErr, &providerErr) {
			retryable = providerErr.Retryable
			if providerErr.Code != "" {
				code = providerErr.Code
			}
		}
		r.store.LogCall(&models.AICallLog{
			Capability:   string(req.Capability),
			RouteName:    route.Name,
			ProviderName: provider.Name,
			Success:      false,
			IsFallback:   idx > 0,
			LatencyMs:    latencyMs,
			ErrorCode:    code,
			ErrorMessage: callErr.Error(),
			RequestMeta:  encodeMeta(req.Metadata),
		})
		attemptErrors = append(attemptErrors, fmt.Errorf("%s: %w", provider.Name, callErr))
		if !retryable {
			break
		}
	}

	return nil, errors.Join(attemptErrors...)
}

func (r *Router) ResolvePrimaryProvider(capability Capability) (*models.AIProvider, *models.AIRoute, error) {
	return r.store.ResolvePrimaryProvider(capability)
}
