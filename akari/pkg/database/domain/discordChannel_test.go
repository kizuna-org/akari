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
	entChNoEdge := &ent.DiscordChannel{
		ID:        "c-no-edge",
		Name:      "channel",
		Type:      discordchannel.TypeGUILD_TEXT,
		CreatedAt: now,
	}

	discordChannel := domain.FromEntDiscordChannel(entChNoEdge)
	if discordChannel == nil {
		t.Fatalf("expected non-nil channel for ent without edges")
	}

	if discordChannel.ID != entChNoEdge.ID {
		t.Fatalf("ID mismatch: got=%v want=%v", discordChannel.ID, entChNoEdge.ID)
	}

	if discordChannel.Guild != nil {
		t.Fatalf("expected non-nil Guild struct even when edge missing")
	}

	entGuild := &ent.DiscordGuild{ID: "g-1"}
	entChWithEdge := &ent.DiscordChannel{
		ID:        "c-1",
		Name:      "channel-1",
		Type:      discordchannel.TypeGUILD_TEXT,
		CreatedAt: now,
		Edges:     ent.DiscordChannelEdges{Guild: entGuild},
	}

	discordChannelWithEdge := domain.FromEntDiscordChannel(entChWithEdge)
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
