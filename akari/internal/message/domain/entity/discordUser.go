package entity

import (
	"time"

	databaseDomain "github.com/kizuna-org/akari/pkg/database/domain"
)

type DiscordUser struct {
	ID       string
	Username string
	Bot      bool
}

func (u *DiscordUser) ToDatabaseDiscordUser() databaseDomain.DiscordUser {
	return databaseDomain.DiscordUser{
		ID:        u.ID,
		Username:  u.Username,
		Bot:       u.Bot,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func ToDiscordUser(user *databaseDomain.DiscordUser) *DiscordUser {
	if user == nil {
		return nil
	}

	return &DiscordUser{
		ID:       user.ID,
		Username: user.Username,
		Bot:      user.Bot,
	}
}
