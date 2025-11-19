package domain

//go:generate go tool mockgen -package=mock -source=discordChannel.go -destination=mock/discordChannel.go

import (
	"context"
	"errors"
	"time"

	"github.com/kizuna-org/akari/gen/ent"
)

type DiscordChannelType string

const (
	DiscordChannelTypeGuildText          DiscordChannelType = "GUILD_TEXT"
	DiscordChannelTypeDM                 DiscordChannelType = "DM"
	DiscordChannelTypeGuildVoice         DiscordChannelType = "GUILD_VOICE"
	DiscordChannelTypeGroupDM            DiscordChannelType = "GROUP_DM"
	DiscordChannelTypeGuildCategory      DiscordChannelType = "GUILD_CATEGORY"
	DiscordChannelTypeGuildAnnouncement  DiscordChannelType = "GUILD_ANNOUNCEMENT"
	DiscordChannelTypeAnnouncementThread DiscordChannelType = "ANNOUNCEMENT_THREAD"
	DiscordChannelTypePublicThread       DiscordChannelType = "PUBLIC_THREAD"
	DiscordChannelTypePrivateThread      DiscordChannelType = "PRIVATE_THREAD"
	DiscordChannelTypeGuildStageVoice    DiscordChannelType = "GUILD_STAGE_VOICE"
	DiscordChannelTypeGuildDirectory     DiscordChannelType = "GUILD_DIRECTORY"
	DiscordChannelTypeGuildForum         DiscordChannelType = "GUILD_FORUM"
	DiscordChannelTypeGuildMedia         DiscordChannelType = "GUILD_MEDIA"
)

type DiscordChannelRepository interface {
	CreateDiscordChannel(ctx context.Context, params DiscordChannel) (*DiscordChannel, error)
	GetDiscordChannelByID(ctx context.Context, id string) (*DiscordChannel, error)
	GetDiscordChannelByMessageID(ctx context.Context, messageID string) (*DiscordChannel, error)
	GetDiscordChannelsByGuildID(ctx context.Context, guildID string) ([]*DiscordChannel, error)
	DeleteDiscordChannel(ctx context.Context, id string) error
}

type DiscordChannel struct {
	ID        string
	Type      DiscordChannelType
	Name      string
	GuildID   string
	CreatedAt time.Time
}

func FromEntDiscordChannel(entDiscordChannel *ent.DiscordChannel) (*DiscordChannel, error) {
	if entDiscordChannel == nil {
		return nil, errors.New("discordChannel is nil")
	}

	if entDiscordChannel.Edges.Guild == nil {
		return nil, errors.New("discordChannel.Guild edge is nil")
	}

	discordGuildID := entDiscordChannel.Edges.Guild.ID

	return &DiscordChannel{
		ID:        entDiscordChannel.ID,
		Type:      DiscordChannelType(entDiscordChannel.Type),
		Name:      entDiscordChannel.Name,
		GuildID:   discordGuildID,
		CreatedAt: entDiscordChannel.CreatedAt,
	}, nil
}
