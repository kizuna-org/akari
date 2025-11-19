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

	discordChannel := domain.FromEntDiscordChannel(entChannelWithoutEdge)
	if discordChannel == nil {
		t.Fatalf("expected non-nil channel for ent without edges")
	}

	if discordChannel.ID != entChannelWithoutEdge.ID {
		t.Fatalf("ID mismatch: got=%v want=%v", discordChannel.ID, entChannelWithoutEdge.ID)
	}

	if discordChannel.Guild != nil {
		t.Fatalf("expected non-nil Guild struct even when edge missing")
	}

	entGuild := &ent.DiscordGuild{ID: "guild-id"}
	entChannelWithEdge := &ent.DiscordChannel{
		ID:        "channel-id",
		Name:      "channel-name",
		Type:      discordchannel.TypeGUILD_TEXT,
		CreatedAt: now,
		Edges:     ent.DiscordChannelEdges{Guild: entGuild},
	}

	discordChannelWithEdge := domain.FromEntDiscordChannel(entChannelWithEdge)
	if discordChannelWithEdge == nil {
		t.Fatalf("expected non-nil channel for ent with guild edge")
	}

	if discordChannelWithEdge.Guild.ID != entGuild.ID {
		t.Fatalf("GuildID mismatch: got=%v want=%v", discordChannelWithEdge.Guild.ID, entGuild.ID)
	}
}

func TestFromEntDiscordChannel_Nil(t *testing.T) {
	t.Parallel()

	if domain.FromEntDiscordChannel(nil) != nil {
		t.Fatalf("expected nil when input is nil")
	}
}
