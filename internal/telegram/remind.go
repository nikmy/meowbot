package telegram

import (
	"context"
	"time"

	"github.com/nikmy/meowbot/internal/repo"
	"github.com/nikmy/meowbot/pkg/errors"
	"gopkg.in/telebot.v3"
)

const (
	pubSubChannel = "telegram"
)

func (b *Bot) remind(ctx context.Context, msg *telebot.Message, remindAt time.Time) (string, error) {
	rData := Data{
		MessageID:  msg.ID,
		RemindTime: remindAt,
	}

	id, err := b.repo.Create(ctx, rData, remindAt, []string{pubSubChannel})
	return id, errors.WrapFail(err, "create message reminder")
}

func (b *Bot) bindStatusMsg(ctx context.Context, id string, statusMsg *telebot.Message) error {
	r, err := b.repo.Get(ctx, id)
	if err != nil {
		return err
	}

	data := r.Data.(Data)
	data.StatusMsg = statusMsg

	upd, err := b.repo.Update(ctx, id, data, &r.RemindAt)
	if err != nil {
		return errors.WrapFail(err, "bind status message")
	}

	if !upd {
		b.logger.Infof("no reminder for bind status to")
	}

	return nil
}

func (b *Bot) updateReminderTime(ctx context.Context, id string, newRemindTime time.Time) error {
	r, err := b.repo.Get(ctx, id)
	if err != nil {
		return err
	}

	upd, err := b.repo.Update(ctx, id, r.Data, &newRemindTime)
	if err != nil {
		return errors.WrapFail(err, "change remind time")
	}

	if !upd {
		b.logger.Infof("no reminder to update")
	}

	return nil
}

func (b *Bot) sendReminder(r repo.Reminder) error {
	data := r.Data.(Data)

	_ = b.Delete(data.StatusMsg)

	keyboard := &telebot.ReplyMarkup{InlineKeyboard: [][]telebot.InlineButton{
		{
			telebot.InlineButton{
				Text: "Отложить",
				Data: r.Unique,
			},
			telebot.InlineButton{
				Text: "Завершить",
				Data: r.Unique,
			},
		},
	}}

	for i := range keyboard.InlineKeyboard[0] {
		keyboard.InlineKeyboard[0][i].Unique = r.Unique // will be "\f<callback_name>|<data>"
		b.Handle("unique", nil)
	}

	msg := map[string]any{
		"message":         &telebot.Message{Text: "meow!"},
		"reply_to_msg_id": data.MessageID,
		"reply_markup":    keyboard,
	}

	_, err := b.Raw("sendMessage", msg)
	return err
}
