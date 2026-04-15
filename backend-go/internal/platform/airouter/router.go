package airouter

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	otelCodes "go.opentelemetry.io/otel/codes"
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
			ProviderTypeOllama:           NewOpenAICompatibleClient(),
		},
	}
}

func NewRouterWithStore(store *Store) *Router {
	return &Router{
		store: store,
		clients: map[string]ProviderClient{
			ProviderTypeOpenAICompatible: NewOpenAICompatibleClient(),
			ProviderTypeOllama:           NewOpenAICompatibleClient(),
		},
	}
}

func (r *Router) RegisterClient(providerType string, client ProviderClient) {
	if client == nil {
		return
	}
	r.clients[providerType] = client
}

func (r *Router) Chat(ctx context.Context, req ChatRequest) (result *ChatResult, err error) {
	ctx, span := otel.Tracer("rss-reader-backend").Start(ctx, "Router.Chat")
	defer span.End()
	defer func() {
		if err != nil {
			span.SetStatus(otelCodes.Error, "error")
			span.RecordError(err)
		}
	}()
	/*line backend-go/internal/platform/airouter/router.go:47:2*/ var promptParts []string
	for _, m := range req.Messages {
		promptParts = append(promptParts, m.Content)
	}
	prompt := concatPrompt(promptParts)
	if len(prompt) > 2000 {
		prompt = prompt[:2000]
	}

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
		code := "provider_error"
		if errors.As(callErr, &providerErr) {
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
	}

	finalErr := errors.Join(attemptErrors...)

	return nil, finalErr
}

func (r *Router) ResolvePrimaryProvider(capability Capability) (*models.AIProvider, *models.AIRoute, error) {
	return r.store.ResolvePrimaryProvider(capability)
}

func (r *Router) Embed(ctx context.Context, req EmbeddingRequest, capability Capability) (result *EmbeddingResult, err error) {
	route, providers, err := r.store.LoadRouteWithProviders(capability)
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
		res, callErr := client.Embed(ctx, provider, req)
		latencyMs := int(time.Since(start).Milliseconds())
		if callErr == nil {
			r.store.LogCall(&models.AICallLog{
				Capability:   string(capability),
				RouteName:    route.Name,
				ProviderName: provider.Name,
				Success:      true,
				IsFallback:   idx > 0,
				LatencyMs:    latencyMs,
				RequestMeta:  encodeMeta(req.Metadata),
			})
			return res, nil
		}

		providerErr := &ProviderError{}
		code := "provider_error"
		if errors.As(callErr, &providerErr) {
			if providerErr.Code != "" {
				code = providerErr.Code
			}
		}
		r.store.LogCall(&models.AICallLog{
			Capability:   string(capability),
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
	}

	return nil, errors.Join(attemptErrors...)
}

func concatPrompt(parts []string) string {
	var result string
	for i, p := range parts {
		if i > 0 {
			result += "\n"
		}
		result += p
	}
	return result
}
