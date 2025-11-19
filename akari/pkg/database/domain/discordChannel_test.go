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

	entGuild := &ent.DiscordGuild{ID: "guild-id"}
	entChannelWithEdge := &ent.DiscordChannel{
		ID:        "channel-id",
		Name:      "channel-name",
		Type:      discordchannel.TypeGUILD_TEXT,
		CreatedAt: now,
		Edges:     ent.DiscordChannelEdges{Guild: entGuild},
	}

	tests := []struct {
		name    string
		input   *ent.DiscordChannel
		wantErr bool
	}{
		{name: "missing guild edge", input: entChannelWithoutEdge, wantErr: true},
		{name: "with guild edge", input: entChannelWithEdge, wantErr: false},
		{name: "nil input", input: nil, wantErr: true},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, err := domain.FromEntDiscordChannel(testCase.input)
			if (err != nil) != testCase.wantErr {
				t.Fatalf("unexpected error state: %v", err)
			}

			if testCase.wantErr {
				if got != nil {
					t.Fatalf("expected nil on error, got: %+v", got)
				}

				return
			}

			if got == nil {
				t.Fatalf("expected non-nil result")
			}

			if got.GuildID != entGuild.ID {
				t.Fatalf("GuildID mismatch: got=%v want=%v", got.GuildID, entGuild.ID)
			}
		})
	}
}
