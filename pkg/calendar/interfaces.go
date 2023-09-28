package calendar

import "gopkg.in/telebot.v3"

type inlineBinder interface {
	BindInline(button *telebot.InlineButton, s string) error
}
