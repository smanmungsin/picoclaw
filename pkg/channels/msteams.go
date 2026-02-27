import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

package channels

import (
	"context"
	"fmt"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
)

type MSTeamsChannel struct {
	*BaseChannel
	config      config.MSTeamsConfig
	// client     *teams.Client // TODO: Add actual Teams SDK client
	ctx         context.Context
	cancel      context.CancelFunc
	// Add more fields as needed (e.g., botUserID, teamID, etc.)
}

func NewMSTeamsChannel(cfg config.MSTeamsConfig, messageBus *bus.MessageBus) (*MSTeamsChannel, error) {
   base := NewBaseChannel("msteams", cfg, messageBus, cfg.AllowFrom)
   return &MSTeamsChannel{
	   BaseChannel: base,
	   config:      cfg,
	   // client:      teams.NewClient(cfg.AppID, cfg.AppSecret, cfg.TenantID), // Example placeholder
   }, nil
}

func (c *MSTeamsChannel) Start(ctx context.Context) error {
	// TODO: Initialize connection to MS Teams, authenticate, and start event loop
	c.ctx, c.cancel = context.WithCancel(ctx)
	c.setRunning(true)
	go c.eventLoop()
	return nil
}

func (c *MSTeamsChannel) Stop(ctx context.Context) error {
	// Clean up resources, stop listeners
	if c.cancel != nil {
		 c.cancel()
	}
	c.setRunning(false)
	return nil
}

func (c *MSTeamsChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return fmt.Errorf("MS Teams channel not running")
	}
	// Example: Send a message using Microsoft Graph API
	// You must implement OAuth2 authentication and token management for production use.
	// This is a simplified example using a placeholder access token.
	accessToken := "YOUR_ACCESS_TOKEN" // TODO: Replace with real token management
	channelID, threadID := parseTeamsChatID(msg.ChatID)
	apiURL := fmt.Sprintf("https://graph.microsoft.com/v1.0/teams/%s/channels/%s/messages", c.config.TeamID, channelID)
	body := map[string]interface{}{
		"body": map[string]string{"content": msg.Content},
	}
	jsonBody, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		data, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("failed to send Teams message: %s", string(data))
	}
	return nil
}

func (c *MSTeamsChannel) eventLoop() {
	// Example: Poll for new messages every 10 seconds (replace with webhook or real-time API for production)
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			// TODO: Implement polling logic using Microsoft Graph API
		}
	}
}
// parseTeamsChatID parses Teams chat/thread IDs
func parseTeamsChatID(chatID string) (channelID, threadID string) {
	// Example: "channelID/threadID" or just "channelID"
	parts := strings.SplitN(chatID, "/", 2)
	channelID = parts[0]
	if len(parts) > 1 {
		threadID = parts[1]
	}
	return channelID, threadID
}
