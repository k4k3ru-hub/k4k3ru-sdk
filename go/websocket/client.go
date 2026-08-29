package websocket

import (
	"context"
	"fmt"

	k4k3ruWebSocket "github.com/k4k3ru-hub/websocket/go"
)

type physicalClient interface {
	Connect(context.Context) error
	Disconnect() error
	Close() error
	SendRaw(context.Context, []byte) error
	Subscribe(context.Context, string, []byte) error
	Unsubscribe(context.Context, string, []byte) error
}

type clientFactory interface {
	New(context.Context, string, k4k3ruWebSocket.SessionHandler, *k4k3ruWebSocket.ClientOption) (physicalClient, error)
}

type physicalClientFactory struct{}

func (physicalClientFactory) New(ctx context.Context, endpointURL string, handler k4k3ruWebSocket.SessionHandler, option *k4k3ruWebSocket.ClientOption) (physicalClient, error) {
	client, err := k4k3ruWebSocket.NewClient(ctx, endpointURL, handler, option)
	if err != nil {
		return nil, fmt.Errorf("failed to create websocket physical client: %w", err)
	}
	return client, nil
}

type client struct {
	physical physicalClient
}

func (c *client) close() error {
	if c == nil {
		return fmt.Errorf("failed to close websocket client: client=null")
	}
	if c.physical == nil {
		return fmt.Errorf("failed to close websocket client: physical_client=null")
	}
	if err := c.physical.Close(); err != nil {
		return fmt.Errorf("failed to close websocket client: %w", err)
	}
	return nil
}

func (c *client) disconnect() error {
	if c == nil {
		return fmt.Errorf("failed to disconnect websocket client: client=null")
	}
	if c.physical == nil {
		return fmt.Errorf("failed to disconnect websocket client: physical_client=null")
	}
	if err := c.physical.Disconnect(); err != nil {
		return fmt.Errorf("failed to disconnect websocket client: %w", err)
	}
	return nil
}

func (c *client) sendRaw(ctx context.Context, payload []byte) error {
	if c == nil {
		return fmt.Errorf("failed to send websocket message: client=null")
	}
	if c.physical == nil {
		return fmt.Errorf("failed to send websocket message: physical_client=null")
	}
	if err := c.physical.SendRaw(ctx, payload); err != nil {
		return fmt.Errorf("failed to send websocket message: %w", err)
	}
	return nil
}
