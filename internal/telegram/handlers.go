package telegram

import (
	"gopkg.in/telebot.v3"
	"strings"
)

func (b *Bot) createInterview(ctx telebot.Context) error {
	args := strings.Split(ctx.Message().Text, " ")
	if len(args) < 3 {
		return ctx.Send("usage: /new @interviewer @candidate")
	}

}
