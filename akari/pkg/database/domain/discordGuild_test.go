package domain_test

import (
	"testing"
	"time"

	"github.com/kizuna-org/akari/gen/ent"
	"github.com/kizuna-org/akari/pkg/database/domain"
)

func TestFromEntDiscordGuild_ChannelIDsNil(t *testing.T) {
	t.Parallel()

	now := time.Now()
	entDiscordGuild := &ent.DiscordGuild{ID: "12345", Name: "Guild1", CreatedAt: now}

	discordGuild := domain.FromEntDiscordGuild(entDiscordGuild)
	if discordGuild == nil {
		t.Fatalf("expected non-nil domain guild")
	}

	if len(discordGuild.Channels) != 0 {
		t.Fatalf("expected empty Channels when edges not loaded, got: %+v", discordGuild.Channels)
	}
}

func TestFromEntDiscordGuild_Nil(t *testing.T) {
	t.Parallel()

	if domain.FromEntDiscordGuild(nil) != nil {
		t.Fatalf("expected nil when input is nil")
	}
}

func TestFromEntDiscordGuild_IncludesChannels(t *testing.T) {
	t.Parallel()

	now := time.Now()
	entChannel := &ent.DiscordChannel{ID: "c1", Name: "Channel1", CreatedAt: now}
	entDiscordGuild := &ent.DiscordGuild{
		ID:        "12345",
		Name:      "Guild1",
		CreatedAt: now,
		Edges: ent.DiscordGuildEdges{
			Channels: []*ent.DiscordChannel{entChannel},
		},
	}

	discordGuild := domain.FromEntDiscordGuild(entDiscordGuild)
	if discordGuild == nil {
		t.Fatalf("expected non-nil domain guild")
	}

	if len(discordGuild.Channels) != 1 {
		t.Fatalf("expected 1 Channel converted, got=%d", len(discordGuild.Channels))
	}

	ch := discordGuild.Channels[0]
	if ch.ID != entChannel.ID || ch.Name != entChannel.Name {
		t.Fatalf("converted Channel fields mismatch: got=%+v want id=%s name=%s", ch, entChannel.ID, entChannel.Name)
	}
}
