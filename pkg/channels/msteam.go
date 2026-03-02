package channels

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/sipeed/picoclaw/pkg/bus"
)

// MSTeamsChannel provides integration with Microsoft Teams
// Implements basic send/receive for messages

type MSTeamsChannel struct {
	WebhookURL string
	running    bool
}

func NewMSTeamsChannel(webhookURL string) *MSTeamsChannel {
	return &MSTeamsChannel{WebhookURL: webhookURL, running: false}
}

func (c *MSTeamsChannel) Name() string {
	return "msteam"
}

func (c *MSTeamsChannel) Start(ctx context.Context) error {
	c.running = true
	return nil
}

func (c *MSTeamsChannel) Stop(ctx context.Context) error {
	c.running = false
	return nil
}

func (c *MSTeamsChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	payload := map[string]string{"text": msg.Content}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	resp, err := http.Post(c.WebhookURL, "application/json", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("MS Teams send failed: %s", resp.Status)
	}
	return nil
}

func (c *MSTeamsChannel) IsRunning() bool {
	return c.running
}

func (c *MSTeamsChannel) IsAllowed(senderID string) bool {
	// Allow all senders for now
	return true
}

// ReceiveMessage is a stub for receiving messages from MS Teams
func (c *MSTeamsChannel) ReceiveMessage() (string, error) {
	// Receiving requires a webhook endpoint and Teams connector setup
	// This is a stub for future implementation
	return "", nil
}
