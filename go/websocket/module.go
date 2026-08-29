package websocket

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	k4k3ruSDKAuthentication "github.com/k4k3ru-hub/k4k3ru-sdk/go/authentication"
	k4k3ruWebSocket "github.com/k4k3ru-hub/websocket/go"
)

// ModuleConfig contains K4K3RU WebSocket endpoint and connection settings.
type ModuleConfig struct {
	EndpointURL        string
	ConnectTimeout     time.Duration
	HandshakeTimeout   time.Duration
	CredentialProvider k4k3ruSDKAuthentication.CredentialProvider
}

// Module owns the composed K4K3RU WebSocket client graph.
type Module struct {
	client        *client
	requests      *requestTracker
	events        *aggregationEventRegistry
	router        *messageRouter
	subscriptions *subscriptionLifecycle
	aggregation   *AggregationClient
}

// NewModule composes a K4K3RU WebSocket module.
//
// Parameters:
//   - ctx: module lifetime context.
//   - config: endpoint and connection settings.
//
// Returns:
//   - Composed WebSocket module.
//   - Configuration or composition error.
//
// Version:
//   - 2026-08-29: Composed authentication and the aggregation subscription client.
//   - 2026-08-29: Added.
func NewModule(ctx context.Context, config ModuleConfig) (*Module, error) {
	return newModule(ctx, config, moduleDeps{
		ClientFactory: physicalClientFactory{},
	})
}

// Close closes the physical WebSocket client owned by the module.
//
// Returns:
//   - Close error.
//
// Version:
//   - 2026-08-29: Added.
func (m *Module) Close() error {
	if m == nil {
		return fmt.Errorf("failed to close websocket module: module=null")
	}
	if m.client == nil {
		return fmt.Errorf("failed to close websocket module: client=null")
	}
	if m.router != nil {
		m.router.HandleClose()
	}
	if err := m.client.close(); err != nil {
		return fmt.Errorf("failed to close websocket module: %w", err)
	}
	return nil
}

type moduleDeps struct {
	ClientFactory clientFactory
}

func newModule(ctx context.Context, config ModuleConfig, deps moduleDeps) (*Module, error) {
	if ctx == nil {
		return nil, fmt.Errorf("failed to create websocket module: context=null")
	}
	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("failed to create websocket module: %w", err)
	}
	if deps.ClientFactory == nil {
		return nil, fmt.Errorf("failed to create websocket module: client_factory=null")
	}
	requests := newRequestTracker()
	events := newAggregationEventRegistry()
	router, err := newMessageRouter(requests, events)
	if err != nil {
		return nil, fmt.Errorf("failed to create websocket module: %w", err)
	}
	handler := &sessionHandler{receiver: router}
	option := k4k3ruWebSocket.DefaultClientOption()
	option.ConnectTimeout = config.ConnectTimeout
	option.HandshakeTimeout = config.HandshakeTimeout
	physicalClient, err := deps.ClientFactory.New(ctx, config.EndpointURL, handler, option)
	if err != nil {
		return nil, fmt.Errorf("failed to create websocket module: failed to create physical client: %w", err)
	}
	if physicalClient == nil {
		return nil, fmt.Errorf("failed to create websocket module: physical_client=null")
	}
	transportClient := &client{physical: physicalClient}
	subscriptions, err := newSubscriptionLifecycle(transportClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create websocket module: %w", err)
	}
	sender, err := newRequestSender(transportClient, requests, config.CredentialProvider, systemClock{}, secureNonceGenerator{}, &atomicRequestIDGenerator{})
	if err != nil {
		return nil, fmt.Errorf("failed to create websocket module: %w", err)
	}
	aggregationClient, err := newAggregationClient(sender, subscriptions, events)
	if err != nil {
		return nil, fmt.Errorf("failed to create websocket module: %w", err)
	}
	return &Module{
		client:        transportClient,
		requests:      requests,
		events:        events,
		router:        router,
		subscriptions: subscriptions,
		aggregation:   aggregationClient,
	}, nil
}

// Aggregation returns the composed Market Hub aggregation WebSocket client.
//
// Returns:
//   - Aggregation client, or nil for a nil or incomplete module.
//
// Version:
//   - 2026-08-29: Added.
func (m *Module) Aggregation() *AggregationClient {
	if m == nil {
		return nil
	}
	return m.aggregation
}

func (c ModuleConfig) validate() error {
	if c.EndpointURL == "" {
		return fmt.Errorf("failed to validate websocket module config: endpoint_url=empty")
	}
	if c.ConnectTimeout < 0 {
		return fmt.Errorf("failed to validate websocket module config: connect_timeout=out_of_range")
	}
	if c.HandshakeTimeout < 0 {
		return fmt.Errorf("failed to validate websocket module config: handshake_timeout=out_of_range")
	}
	if c.CredentialProvider == nil {
		return fmt.Errorf("failed to validate websocket module config: credential_provider=null")
	}
	endpoint, err := url.Parse(c.EndpointURL)
	if err != nil {
		return fmt.Errorf("failed to validate websocket module config: failed to parse endpoint url: %w: endpoint_url=%q", err, c.EndpointURL)
	}
	if endpoint.Scheme != "ws" && endpoint.Scheme != "wss" {
		return fmt.Errorf("failed to validate websocket module config: endpoint_url=invalid")
	}
	if endpoint.Host == "" {
		return fmt.Errorf("failed to validate websocket module config: endpoint_url=invalid")
	}
	if endpoint.User != nil || endpoint.RawQuery != "" || endpoint.Fragment != "" {
		return fmt.Errorf("failed to validate websocket module config: endpoint_url=invalid")
	}
	if endpoint.Scheme == "ws" && !isLoopbackHostname(endpoint.Hostname()) {
		return fmt.Errorf("failed to validate websocket module config: endpoint_url=invalid")
	}
	return nil
}

func isLoopbackHostname(hostname string) bool {
	if strings.EqualFold(hostname, "localhost") {
		return true
	}
	address := net.ParseIP(hostname)
	return address != nil && address.IsLoopback()
}
