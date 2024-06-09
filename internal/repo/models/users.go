package models

import (
	"context"
	"strconv"
)

type UsersRepo interface {
	Update(ctx context.Context, username string, telegramID *int64, category *UserCategory, intGrade *int) (*User, error)
	Upsert(ctx context.Context, username string, telegramID *int64, category *UserCategory, intGrade *int) (*User, error)
	Get(ctx context.Context, username string) (*User, error)

	UpdateMeetings(ctx context.Context, username string, meets []Meeting, old []Meeting) (bool, error)
	Match(ctx context.Context, targetInterval [2]int64) ([]User, error)
}

type User struct {
	Telegram int64        `json:"telegram" bson:"telegram"`
	Assigned []Meeting    `json:"assigned" bson:"assigned"`
	Username string       `json:"username" bson:"username"`
	Category UserCategory `json:"category" bson:"category"`
	IntGrade int          `json:"intGrade" bson:"intGrade"`
}

func (u User) Recipient() string {
	if u.Telegram == 0 {
		return ""
	}

	return strconv.FormatInt(u.Telegram, 10)
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

type Meeting [2]int64

const (
	UserFieldUsername = "username"
	UserFieldTelegram = "telegram"
	UserFieldAssigned = "assigned"
	UserFieldCategory = "category"
	UserFieldIntGrade = "intGrade"
)
