package models

import (
	"context"
	"strconv"
)

type UsersRepo interface {
	Upsert(ctx context.Context, username string, telegramID *int64, category *UserCategory, intGrade *int) error
	Get(ctx context.Context, username string) (*User, error)

	Match(ctx context.Context, targetInterval [2]int64) ([]User, error)
	Schedule(ctx context.Context, username string, meeting Meeting) (bool, error)
	Free(ctx context.Context, username string, meeting Meeting) error
}

type User struct {
	Telegram int64        `json:"telegram" bson:"telegram"`
	Meetings []Meeting    `json:"assigned" bson:"assigned"`
	Username string       `json:"username" bson:"username"`
	Category UserCategory `json:"category" bson:"category"`
	IntGrade int         `json:"intGrade" bson:"intGrade"`
}

const (
	GradeNotInterviewer int = 0
)

type UserCategory int

const (
	ExternalUser UserCategory = iota
	EmployeeUser
	HRUser
)

func (u User) Recipient() string {
	if u.Telegram == 0 {
		return ""
	}

	return strconv.FormatInt(u.Telegram, 10)
}

type Meeting [2]int64

const (
	UserFieldUsername = "username"
	UserFieldTelegram = "telegram"
	UserFieldMeetings = "meetings"
	UserFieldCategory = "category"
	UserFieldIntGrade = "intGrade"
)
