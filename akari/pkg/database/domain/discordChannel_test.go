package domain_test

import (
	"testing"
	"time"

	"github.com/kizuna-org/akari/gen/ent"
	"github.com/kizuna-org/akari/gen/ent/discordchannel"
	"github.com/kizuna-org/akari/pkg/database/domain"
)

func TestFromEntDiscordChannel_NilAndEdges(t *testing.T) {
	t.Parallel()

	now := time.Now()
	entChannelWithoutEdge := &ent.DiscordChannel{
		ID:        "channel-id",
		Name:      "channel-name",
		Type:      discordchannel.TypeGUILD_TEXT,
		CreatedAt: now,
	}

	discordChannel, err := domain.FromEntDiscordChannel(entChannelWithoutEdge)
	if err == nil {
		t.Fatalf("expected error when Guild edge is nil")
	}

	if discordChannel != nil {
		t.Fatalf("expected nil channel for ent without edges")
	}

	entGuild := &ent.DiscordGuild{ID: "guild-id"}
	entChannelWithEdge := &ent.DiscordChannel{
		ID:        "channel-id",
		Name:      "channel-name",
		Type:      discordchannel.TypeGUILD_TEXT,
		CreatedAt: now,
		Edges:     ent.DiscordChannelEdges{Guild: entGuild},
	}

	discordChannelWithEdge, err := domain.FromEntDiscordChannel(entChannelWithEdge)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if discordChannelWithEdge == nil {
		t.Fatalf("expected non-nil channel for ent with guild edge")
	}

	if discordChannelWithEdge.GuildID != entGuild.ID {
		t.Fatalf("GuildID mismatch: got=%v want=%v", discordChannelWithEdge.GuildID, entGuild.ID)
	}
}

func TestFromEntDiscordChannel_Nil(t *testing.T) {
	t.Parallel()

	discordChannel, err := domain.FromEntDiscordChannel(nil)
	if err == nil {
		t.Fatalf("expected error when input is nil")
	}

	if discordChannel != nil {
		t.Fatalf("expected nil channel when input is nil")
	}
}
