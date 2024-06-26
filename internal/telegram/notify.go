package telegram

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strconv"
	"time"

	"gopkg.in/telebot.v3"

	"github.com/nikmy/meowbot/internal/repo/models"
	"github.com/nikmy/meowbot/pkg/errors"
	"github.com/nikmy/meowbot/pkg/txn"
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
	Interview  *models.Interview
	Recipients []models.Role
	NotifyTime int64
	LeftTime   time.Duration
}

func (b *Bot) sendNeededNotifications() error {
	now := time.Now().Add(time.Hour * 3).UnixMilli()
	fut := now + b.notifyBefore[len(b.notifyBefore)-1]
	prv := now - time.Minute.Milliseconds()

	upcoming, err := b.repo.Interviews().GetUpcoming(b.ctx, prv, fut)
	if err != nil {
		return errors.WrapFail(err, "get ready interviews")
	}

	needed := b.getNeededNotifications(now, upcoming)
	if len(needed) == 0 {
		return nil
	}

	b.sendAllNotifications(needed)

	b.log.Debugf("sent %d", len(needed))

	return nil
}

func (b *Bot) sendAllNotifications(ns []notification) {
	ctx, cancel, err := b.txm.NewSessionContext(b.ctx, b.notifyPeriod)
	if err != nil {
		b.log.Error(errors.WrapFail(err, "create session context"))
		return
	}
	defer cancel()

	for _, n := range ns {
		if b.sendOneNotification(ctx, n.Interview.InterviewerTg, n, [2]bool{true}) {
			b.sendOneNotification(ctx, n.Interview.CandidateTg, n, [2]bool{true, true})
		}
	}
}

func (b *Bot) sendOneNotification(ctx context.Context, tgID int64, n notification, roles [2]bool) bool {
	var msg string
	if n.LeftTime == 0 {
		msg = fmt.Sprintf(
			"Собеседование %s вот-вот начнётся! Подключиться можно по ссылке %s\nУдачи!",
			n.Interview.ID, n.Interview.Zoom,
		)
	} else {
		var left string
		weeks, days := int(n.LeftTime.Hours()/168), int(n.LeftTime.Hours()/24)
		switch {
		case weeks > 2:
			left = strconv.Itoa(weeks) + " нед."
		case days > 2:
			left = strconv.Itoa(days) + " д."
		case n.LeftTime.Hours() > 1:
			left = strconv.Itoa(int(n.LeftTime.Hours())) + " ч."
		default:
			left = strconv.Itoa(int(n.LeftTime.Minutes())) + " мин."
		}

		msg = fmt.Sprintf(
			"До собеседования `%s` на должность \"%s\" осталось менее %s",
			n.Interview.ID, n.Interview.Vacancy, left,
		)
	}

	tx, err := txn.New(ctx).
		SetModel(txn.CausalConsistency).
		SetIsolation(txn.SnapshotIsolation).
		Start(ctx)
	if err != nil {
		b.log.Error(errors.WrapFail(err, "start txn"))
		return false
	}
	defer func() {
		err := tx.Close(ctx)
		if err != nil {
			b.log.Warn(errors.WrapFail(err, "close txn"))
		}
	}()

	err = b.notify(tgID, msg)
	if err != nil {
		b.log.Error(errors.WrapFail(err, "notify user %d", tgID))
		return false
	}

	err = b.repo.Interviews().Notify(ctx, n.Interview.ID, n.NotifyTime, roles)
	if err != nil {
		b.log.Error(errors.WrapFail(err, "do Interviews.Notify request"))
		return false
	}

	err = tx.Commit(ctx)
	if err != nil {
		b.log.Error(errors.WrapFail(err, "commit txn"))
		return false
	}

	return true
}

func (b *Bot) getNeededNotifications(now int64, upcoming []*models.Interview) []notification {
	needed := make([]notification, 0, len(upcoming))

	both := []models.Role{models.RoleInterviewer, models.RoleCandidate}

	for _, i := range upcoming {
		left := i.Meet[0] - now

		// check if interview is almost started
		if left < b.notifyPeriod.Milliseconds() {
			if i.LastNotification == nil || i.LastNotification.UnixTime != i.Meet[0] {
				needed = append(needed, notification{
					Interview:  i,
					Recipients: both,
					NotifyTime: i.Meet[0],
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

		if last == nil || i.Meet[0]-last.UnixTime > chosenInt {
			needed = append(needed, notification{
				Interview:  i,
				Recipients: both,
				NotifyTime: i.Meet[0] - chosenInt,
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
			LeftTime:   time.Duration(i.Meet[0]-last.UnixTime) * time.Millisecond,
		})
	}

	return needed
}
