package telegram

import (
	"fmt"
	"slices"
	"time"

	"gopkg.in/telebot.v3"

	"github.com/nikmy/meowbot/internal/interviews"
	"github.com/nikmy/meowbot/internal/users"
	"github.com/nikmy/meowbot/pkg/errors"
)

func (b *Bot) notify(userID int64, msg string) error {
	_, err := b.bot.Send(users.User{Telegram: userID}, msg, &telebot.SendOptions{ParseMode: telebot.ModeMarkdown})
	return err
}

func (b *Bot) watch() {
	tick := time.NewTicker(b.notifyPeriod)
	defer tick.Stop()

	for {
		select {
		case <-b.ctx.Done():
			return
		case <-tick.C:
			_, err := b.interviews.Txn(b.ctx, b.sendNeededNotifications)
			if err != nil {
				b.log.Error(errors.WrapFail(err, "send needed notifications"))
			}
		}
	}
}

type notification struct {
	Interview  interviews.Interview
	Recipients []interviews.Role
	NotifyTime int64
	LeftTime   time.Duration
}

func (b *Bot) sendNeededNotifications() error {
	now := time.Now().UTC().UnixMilli()
	fut := now + b.notifyBefore[len(b.notifyBefore)-1]

	upcoming, err := b.interviews.GetReadyAt(b.ctx, fut)
	if err != nil {
		return errors.WrapFail(err, "get ready interviews")
	}

	needed := b.getNeededNotifications(now, upcoming)
	if len(needed) == 0 {
		return nil
	}

	b.sendAllNotifications(needed)
	return nil
}

func (b *Bot) sendAllNotifications(ns []notification) {
	for _, n := range ns {
		var notified []interviews.Role

		for _, role := range n.Recipients {
			tgID := int64(0)
			switch role {
			case interviews.RoleInterviewer:
				tgID = n.Interview.InterviewerTg
			case interviews.RoleCandidate:
				tgID = n.Interview.CandidateTg
			}

			msg := fmt.Sprintf(
				"До собеседования `%s` на должность \"%s\" осталось %s",
				n.Interview.ID, n.Interview.Vacancy, n.LeftTime,
			)

			err := b.notify(tgID, msg)
			if err != nil {
				b.log.Error(errors.WrapFail(err, "notify interviewer"))
				continue
			}

			notified = append(notified, role)
		}

		if len(notified) == 0 {
			continue
		}

		err := b.interviews.Notify(b.ctx, n.Interview.ID, n.NotifyTime, notified)
		if err != nil {
			b.log.Error(errors.WrapFail(err, "do Interviews.Notify request"))
		}
	}
}

func (b *Bot) getNeededNotifications(now int64, upcoming []interviews.Interview) []notification {
	needed := make([]notification, 0, len(upcoming))

	for _, i := range upcoming {
		last := i.LastNotification
		left := time.Duration(i.Interval[0]-now) * time.Millisecond

		lastNotifyInterval := i.Interval[0] - last.UnixTime
		firstLarger, _ := slices.BinarySearch(b.notifyBefore, lastNotifyInterval)

		if firstLarger > 0 {
			proximity := lastNotifyInterval - b.notifyBefore[firstLarger-1]
			if proximity < b.notifyPeriod.Milliseconds() {
				needed = append(needed, notification{
					Interview:  i,
					NotifyTime: b.notifyBefore[firstLarger-1] - b.notifyPeriod.Milliseconds(),
					LeftTime:   left,
				})
				continue
			}
		}

		if firstLarger == len(b.notifyBefore) {
			b.log.Debug(errors.Error("wrong case"))
			continue
		}

		proximity := b.notifyBefore[firstLarger] - lastNotifyInterval
		if proximity < b.notifyPeriod.Milliseconds() {
			needed = append(needed, notification{
				Interview:  i,
				NotifyTime: b.notifyBefore[firstLarger] + b.notifyPeriod.Milliseconds(),
				LeftTime:   left,
			})
		}

		if last.Notified[interviews.RoleInterviewer] && last.Notified[interviews.RoleCandidate] {
			continue
		}

		n := notification{
			Interview:  i,
			NotifyTime: last.UnixTime,
			LeftTime:   left,
		}
		if !last.Notified[interviews.RoleInterviewer] {
			n.Recipients = append(n.Recipients, interviews.RoleInterviewer)
		}
		if !last.Notified[interviews.RoleCandidate] {
			n.Recipients = append(n.Recipients, interviews.RoleCandidate)
		}
		needed = append(needed, n)
	}

	return needed
}
