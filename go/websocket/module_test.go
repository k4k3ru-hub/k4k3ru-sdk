package websocket

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	k4k3ruSDKAuthentication "github.com/k4k3ru-hub/k4k3ru-sdk/go/authentication"
	k4k3ruWebSocket "github.com/k4k3ru-hub/websocket/go"
)

func TestNewModuleComposesPhysicalClient(t *testing.T) {
	t.Parallel()

	factory := &fakeClientFactory{client: &fakePhysicalClient{}}
	config := ModuleConfig{
		EndpointURL:        "wss://api.k4k3ru.com/",
		ConnectTimeout:     2 * time.Second,
		HandshakeTimeout:   4 * time.Second,
		CredentialProvider: fakeCredentialProvider{},
	}
	module, err := newModule(context.Background(), config, moduleDeps{ClientFactory: factory})
	if err != nil {
		t.Fatalf("newModule() error = %v", err)
	}
	if module == nil || module.client == nil || module.client.physical != factory.client || module.requests == nil || module.events == nil || module.router == nil || module.subscriptions == nil {
		t.Fatalf("newModule() = %#v", module)
	}
	if factory.endpointURL != config.EndpointURL {
		t.Fatalf("endpoint URL = %q, want %q", factory.endpointURL, config.EndpointURL)
	}
	if factory.option == nil || factory.option.ConnectTimeout != config.ConnectTimeout || factory.option.HandshakeTimeout != config.HandshakeTimeout {
		t.Fatalf("client option = %#v", factory.option)
	}
	if factory.handler == nil {
		t.Fatal("session handler = nil")
	}
	pending, err := module.requests.register([]byte(`1`))
	if err != nil {
		t.Fatalf("register() error = %v", err)
	}
	factory.handler.HandleMessage(nil, []byte(`{"id":1,"result":{"ok":true}}`))
	if outcome := <-pending.outcome; outcome.err != nil || string(outcome.response.Result) != `{"ok":true}` {
		t.Fatalf("outcome = %#v", outcome)
	}
	pending, err = module.requests.register([]byte(`2`))
	if err != nil {
		t.Fatalf("register() error = %v", err)
	}
	factory.handler.HandleClose(nil)
	if outcome := <-pending.outcome; !errors.Is(outcome.err, errWebSocketConnectionClosed) {
		t.Fatalf("close outcome error = %v", outcome.err)
	}
}

func TestNewModuleValidatesDependenciesAndConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		ctx    context.Context
		config ModuleConfig
		deps   moduleDeps
		want   string
	}{
		{name: "nil context", config: validModuleConfig("wss://api.k4k3ru.com/"), deps: validModuleDeps(), want: "context=null"},
		{name: "empty endpoint", ctx: context.Background(), deps: validModuleDeps(), want: "endpoint_url=empty"},
		{name: "insecure endpoint", ctx: context.Background(), config: validModuleConfig("ws://api.k4k3ru.com/"), deps: validModuleDeps(), want: "endpoint_url=invalid"},
		{name: "invalid scheme", ctx: context.Background(), config: validModuleConfig("https://api.k4k3ru.com/"), deps: validModuleDeps(), want: "endpoint_url=invalid"},
		{name: "negative connect timeout", ctx: context.Background(), config: ModuleConfig{EndpointURL: "wss://api.k4k3ru.com/", ConnectTimeout: -time.Second, CredentialProvider: fakeCredentialProvider{}}, deps: validModuleDeps(), want: "connect_timeout=out_of_range"},
		{name: "negative handshake timeout", ctx: context.Background(), config: ModuleConfig{EndpointURL: "wss://api.k4k3ru.com/", HandshakeTimeout: -time.Second, CredentialProvider: fakeCredentialProvider{}}, deps: validModuleDeps(), want: "handshake_timeout=out_of_range"},
		{name: "nil credential provider", ctx: context.Background(), config: ModuleConfig{EndpointURL: "wss://api.k4k3ru.com/"}, deps: validModuleDeps(), want: "credential_provider=null"},
		{name: "nil factory", ctx: context.Background(), config: validModuleConfig("wss://api.k4k3ru.com/"), deps: moduleDeps{}, want: "client_factory=null"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := newModule(test.ctx, test.config, test.deps)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("newModule() error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestModuleClose(t *testing.T) {
	t.Parallel()

	physical := &fakePhysicalClient{}
	requests := newRequestTracker()
	pending, err := requests.register([]byte(`1`))
	if err != nil {
		t.Fatalf("register() error = %v", err)
	}
	router, err := newMessageRouter(requests, newAggregationEventRegistry())
	if err != nil {
		t.Fatalf("newMessageRouter() error = %v", err)
	}
	module := &Module{client: &client{physical: physical}, router: router}
	if err := module.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if !physical.closed {
		t.Fatal("physical client was not closed")
	}
	if outcome := <-pending.outcome; !errors.Is(outcome.err, errWebSocketConnectionClosed) {
		t.Fatalf("close outcome error = %v", outcome.err)
	}

	physical.closeErr = errors.New("close failure")
	err = module.Close()
	if err == nil || !strings.Contains(err.Error(), "close failure") {
		t.Fatalf("Close() error = %v, want close failure", err)
	}
}

func validModuleDeps() moduleDeps {
	return moduleDeps{ClientFactory: &fakeClientFactory{client: &fakePhysicalClient{}}}
}

func validModuleConfig(endpointURL string) ModuleConfig {
	return ModuleConfig{EndpointURL: endpointURL, CredentialProvider: fakeCredentialProvider{}}
}

type fakeCredentialProvider struct{}

func (fakeCredentialProvider) Credential(context.Context) (k4k3ruSDKAuthentication.Credential, error) {
	return k4k3ruSDKAuthentication.Credential{}, nil
}

type fakeClientFactory struct {
	client      *fakePhysicalClient
	endpointURL string
	handler     k4k3ruWebSocket.SessionHandler
	option      *k4k3ruWebSocket.ClientOption
}

func (f *fakeClientFactory) New(_ context.Context, endpointURL string, handler k4k3ruWebSocket.SessionHandler, option *k4k3ruWebSocket.ClientOption) (physicalClient, error) {
	f.endpointURL = endpointURL
	f.handler = handler
	f.option = option
	if f.client == nil {
		f.client = &fakePhysicalClient{}
	}
	return f.client, nil
}

type fakePhysicalClient struct {
	closed        bool
	disconnected  bool
	closeErr      error
	disconnectErr error
}

func (*fakePhysicalClient) Connect(context.Context) error                     { return nil }
func (f *fakePhysicalClient) Disconnect() error                               { f.disconnected = true; return f.disconnectErr }
func (f *fakePhysicalClient) Close() error                                    { f.closed = true; return f.closeErr }
func (*fakePhysicalClient) SendRaw(context.Context, []byte) error             { return nil }
func (*fakePhysicalClient) Subscribe(context.Context, string, []byte) error   { return nil }
func (*fakePhysicalClient) Unsubscribe(context.Context, string, []byte) error { return nil }
