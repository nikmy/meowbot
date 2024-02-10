package telegram

import (
	"time"

	"gopkg.in/telebot.v3"
)

type Data struct {
	// MessageID - сообщение, о котором нужно напомнить
	MessageID int `json:"message" bson:"message"`

	// StatusMsg - сообщение с кнопками и настройкой напоминания
	StatusMsg *telebot.Message `json:"tuneMsg" bson:"tuneMsg"`

	// RemindTime - заданное время отправки напоминания
	RemindTime time.Time `json:"remind_time" bson:"remindTime"`
}
