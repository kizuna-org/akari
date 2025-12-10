package entity

import (
	"time"

	databaseDomain "github.com/kizuna-org/akari/pkg/database/domain"
	discordEntity "github.com/kizuna-org/akari/pkg/discord/domain/entity"
)

type User struct {
	ID       string
	Username string
	Bot      bool
}

func (u *User) ToDatabaseUser() databaseDomain.DiscordUser {
	return databaseDomain.DiscordUser{
		ID:        u.ID,
		Username:  u.Username,
		Bot:       u.Bot,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func ToUser(user *discordEntity.User) *User {
	if user == nil {
		return nil
	}

	return &User{
		ID:       user.ID,
		Username: user.Username,
		Bot:      user.Bot,
	}
}
