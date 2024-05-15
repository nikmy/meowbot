package users

import (
	"strconv"

	"github.com/nikmy/meowbot/internal/interviews"
)

type User struct {
	Telegram int64      `json:"telegram" bson:"telegram"`
	Assigned []Interview `json:"assigned" bson:"assigned"`
	Username string      `json:"username" bson:"username"`
	Employee bool        `json:"employee" bson:"employee"`
}

func (u User) Recipient() string {
	if u.Telegram == 0 {
		return ""
	}

	return strconv.FormatInt(u.Telegram, 10)
}

type Role = interviews.Role

type Interview struct {
	Role     Role     `json:"role"      bson:"role"`
	TimeSlot [2]int64 `json:"time_slot" bson:"time_slot"`
}

func getIntervals(interviews []Interview) [][2]int64 {
	intervals := make([][2]int64, 0, len(interviews))
	for _, i := range interviews {
		intervals = append(intervals, i.TimeSlot)
	}
	return intervals
}
