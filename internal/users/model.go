package users

import (
	"strconv"

	"github.com/nikmy/meowbot/internal/interviews"
)

type User struct {
	Telegram int64     `json:"telegram" bson:"telegram"`
	Meetings []Meeting `json:"assigned" bson:"assigned"`
	Username string    `json:"username" bson:"username"`
	Employee bool      `json:"employee" bson:"employee"`
}

func (u User) Recipient() string {
	if u.Telegram == 0 {
		return ""
	}

	return strconv.FormatInt(u.Telegram, 10)
}

type Role = interviews.Role

type Meeting [2]int64
