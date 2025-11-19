package domain_test

import (
	"testing"
	"time"

	"github.com/kizuna-org/akari/gen/ent"
	"github.com/kizuna-org/akari/pkg/database/domain"
)

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

	tests := []struct {
		name    string
		input   *ent.DiscordGuild
		wantErr bool
	}{
		{name: "channels nil edge", input: &ent.DiscordGuild{ID: "g1", Name: "n1", CreatedAt: now}, wantErr: true},
		{name: "with channels", input: entDiscordGuild, wantErr: false},
		{name: "nil input", input: nil, wantErr: true},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, err := domain.FromEntDiscordGuild(testCase.input)
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

			if len(got.ChannelIDs) != len(testCase.input.Edges.Channels) {
				t.Fatalf("channels length mismatch: got=%d want=%d", len(got.ChannelIDs), len(testCase.input.Edges.Channels))
			}

			for i := range got.ChannelIDs {
				if got.ChannelIDs[i] != testCase.input.Edges.Channels[i].ID {
					t.Fatalf("converted Channel ID mismatch: got=%s want=%s", got.ChannelIDs[i], testCase.input.Edges.Channels[i].ID)
				}
			}
		})
	}
}
