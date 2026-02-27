package channels

import (
	"context"
	"testing"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
)

func TestMSTeamsChannel_BasicLifecycle(t *testing.T) {
	cfg := config.MSTeamsConfig{
		Enabled:   true,
		AppID:     "dummy-app-id",
		AppSecret: "dummy-secret",
		TenantID:  "dummy-tenant",
		AllowFrom: []string{"user1"},
	}
	bus := bus.NewMessageBus()
	ch, err := NewMSTeamsChannel(cfg, bus)
	if err != nil {
		t.Fatalf("failed to create MSTeamsChannel: %v", err)
	}

	ctx := context.Background()
	if err := ch.Start(ctx); err != nil {
		t.Errorf("Start() failed: %v", err)
	}
	if !ch.IsRunning() {
		t.Error("Channel should be running after Start()")
	}
	if err := ch.Stop(ctx); err != nil {
		t.Errorf("Stop() failed: %v", err)
	}
	if ch.IsRunning() {
		t.Error("Channel should not be running after Stop()")
	}
}
