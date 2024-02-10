package main

import (
	"fmt"
	"github.com/nikmy/meowbot/pkg/model"
	"gopkg.in/telebot.v3"
	"unsafe"
)

func main() {
	fmt.Println(((unsafe.Sizeof(telebot.Message{})+
		unsafe.Sizeof(telebot.User{})+
		unsafe.Sizeof(telebot.Chat{}))*
		2 +
		unsafe.Sizeof(reminders.Data{})),
	)
}
