package telegram

import (
	"cmp"
	"fmt"
	"slices"
	"time"

	"gopkg.in/telebot.v3"

	"github.com/nikmy/meowbot/internal/repo/models"
	"github.com/nikmy/meowbot/pkg/errors"
)

func (b *Bot) applyNotifications(cfg Config) {
	notifyBefore := make([]int64, len(cfg.NotifyBefore))
	for i := range cfg.NotifyBefore {
		notifyBefore[i] = cfg.NotifyBefore[i].Milliseconds()
	}

	slices.SortFunc(notifyBefore, cmp.Compare[int64])
	b.notifyBefore = notifyBefore
	b.notifyPeriod = cfg.NotifyPeriod
}

func (b *Bot) runNotifier() {
	if b.notifyPeriod > 0 && len(b.notifyBefore) > 0 {
		go b.watch()
	}
}

func (b *Bot) notify(userID int64, msg string) error {
	_, err := b.bot.Send(models.User{Telegram: userID}, msg, &telebot.SendOptions{ParseMode: telebot.ModeMarkdown})
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
			err := b.sendNeededNotifications()
			if err != nil {
				b.log.Error(errors.WrapFail(err, "send needed notifications"))
			}
		}
	}
}

type notification struct {
	Interview  models.Interview
	Recipients []models.Role
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
		for _, role := range n.Recipients {
			tgID := int64(0)
			switch role {
			case models.RoleInterviewer:
				tgID = n.Interview.InterviewerTg
			case models.RoleCandidate:
				tgID = n.Interview.CandidateTg
			}

			msg := fmt.Sprintf(
				"До собеседования `%s` на должность \"%s\" осталось %s",
				n.Interview.ID, n.Interview.Vacancy, n.LeftTime,
			)

			// TODO: txn
			err := b.notify(tgID, msg)
			if err != nil {
				b.log.Error(errors.WrapFail(err, "notify user %d", tgID))
				continue
			}

			err = b.interviews.Notify(b.ctx, n.Interview.ID, n.NotifyTime, role)
			if err != nil {
				b.log.Error(errors.WrapFail(err, "do Interviews.Notify request"))
			}
		}
	}
}

func (b *Bot) getNeededNotifications(now int64, upcoming []models.Interview) []notification {
	needed := make([]notification, 0, len(upcoming))

	both := []models.Role{models.RoleInterviewer, models.RoleCandidate}

	for _, i := range upcoming {
		left := i.Interval[0] - now

		// check if interview is almost started
		if left < b.notifyPeriod.Milliseconds() {
			if i.LastNotification == nil || i.LastNotification.UnixTime != i.Interval[0] {
				needed = append(needed, notification{
					Interview:  i,
					Recipients: both,
					NotifyTime: i.Interval[0],
					LeftTime:   0,
				})
				continue
			}

			continue
		}

		// find appropriate interval to notify
		chosenNotify, _ := slices.BinarySearch(b.notifyBefore, left)

		// too early to notify
		if chosenNotify == len(b.notifyBefore) {
			b.log.Warnf("early attempt to notify")
			continue
		}

		last := i.LastNotification
		chosenInt := b.notifyBefore[chosenNotify]

		if last == nil || i.Interval[0]-last.UnixTime > chosenInt {
			needed = append(needed, notification{
				Interview:  i,
				Recipients: both,
				NotifyTime: i.Interval[0] - chosenInt,
				LeftTime:   time.Duration(chosenInt) * time.Millisecond,
			})
			continue
		}

		// check if last time both sides were notified
		if last.Notified[models.RoleCandidate] {
			continue
		}

		needed = append(needed, notification{
			Interview:  i,
			NotifyTime: last.UnixTime,
			Recipients: both[1:],
			LeftTime:   time.Duration(i.Interval[0]-last.UnixTime) * time.Millisecond,
		})
	}

	return needed
}
