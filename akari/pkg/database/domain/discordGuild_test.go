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
	entDiscordGuild := &ent.DiscordGuild{ID: "guild-id", Name: "guild-name", CreatedAt: now}

	discordGuild, err := domain.FromEntDiscordGuild(entDiscordGuild)
	if err == nil {
		t.Fatalf("expected error when Channels edge is nil")
	}

	if discordGuild != nil {
		t.Fatalf("expected nil domain guild")
	}
}

func TestFromEntDiscordGuild_Nil(t *testing.T) {
	t.Parallel()

	discordGuild, err := domain.FromEntDiscordGuild(nil)
	if err == nil {
		t.Fatalf("expected error when input is nil")
	}

	if discordGuild != nil {
		t.Fatalf("expected nil domain guild when input is nil")
	}
}

func TestFromEntDiscordGuild_IncludesChannels(t *testing.T) {
	t.Parallel()

	now := time.Now()
	entChannel := &ent.DiscordChannel{ID: "channel-id", Name: "channel-name", CreatedAt: now}
	entDiscordGuild := &ent.DiscordGuild{
		ID:        "guild-id",
		Name:      "guild-name",
		CreatedAt: now,
		Edges: ent.DiscordGuildEdges{
			Channels: []*ent.DiscordChannel{entChannel},
		},
	}

	discordGuild, err := domain.FromEntDiscordGuild(entDiscordGuild)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if discordGuild == nil {
		t.Fatalf("expected non-nil domain guild")
	}

	if len(discordGuild.ChannelIDs) != 1 {
		t.Fatalf("expected 1 Channel converted, got=%d", len(discordGuild.ChannelIDs))
	}

	for i, discordChannel := range discordGuild.ChannelIDs {
		if discordChannel != entDiscordGuild.Edges.Channels[i].ID {
			t.Fatalf("converted Channel ID mismatch: got=%s want=%s", discordChannel, entDiscordGuild.Edges.Channels[i].ID)
		}
	}
}
