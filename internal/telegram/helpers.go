package telegram

import (
	"context"
	"slices"

	"github.com/vitaliy-ukiru/fsm-telebot"
	"gopkg.in/telebot.v3"

	"github.com/nikmy/meowbot/internal/repo/models"
	"github.com/nikmy/meowbot/internal/repo/txn"
	"github.com/nikmy/meowbot/pkg/errors"
)

func (b *Bot) cancelInterview(ctx context.Context, interview *models.Interview, side models.Role) (bool, error) {
	if interview.Meet == nil {
		return false, nil
	}

	err := b.repo.Interviews().Cancel(ctx, interview.ID, side)
	if err != nil {
		return false, errors.WrapFail(err, "do Interviews.Cancel request")
	}

	ok, err := b.cancelMeeting(ctx, interview.InterviewerUN, *interview.Meet)
	if err != nil {
		return false, errors.WrapFail(err, "cancel meeting")
	}
	if !ok {
		return false, nil
	}

	ok, err = b.cancelMeeting(ctx, interview.CandidateUN, *interview.Meet)
	if err != nil {
		return false, errors.WrapFail(err, "cancel meeting")
	}
	if !ok {
		return false, nil
	}

	return true, nil
}

func (b *Bot) cancelMeeting(ctx context.Context, username string, meet models.Meeting) (bool, error) {
	user, err := b.repo.Users().Get(ctx, username)
	if err != nil {
		return false, errors.WrapFail(err, "find user")
	}

	if user == nil {
		return false, nil
	}

	meets, found := user.FindAndDeleteMeeting(meet)
	if !found {
		return false, nil
	}

	updated, err := b.repo.Users().UpdateMeetings(ctx, username, meets)
	if err != nil {
		return false, errors.WrapFail(err, "update meetings")
	}

	return updated, nil
}

func (b *Bot) denyNotHR(c telebot.Context, s fsm.Context) error {
	return b.final(c, s, "Это может сделать только HR сотрудник")
}

func (b *Bot) checkHR(username string) bool {
	user, err := b.repo.Users().Get(b.ctx, username)
	if err != nil {
		b.log.Warn(errors.WrapFail(err, "get user for checking HR permissions"))
		return false
	}

	return user.Category >= models.HRUser
}

func (b *Bot) readTg(c telebot.Context) (string, string) {
	tg := c.Text()
	if len(tg) < 2 || tg[0] != '@' {
		return "", "Некорректный telegram"
	}
	return tg[1:], ""
}

func (b *Bot) tryAssign(
	ctx context.Context,
	candidate models.User,
	interviewer models.User,
	iid string,
	meet models.Meeting,
) (bool, bool) {
	if candidate.Username == interviewer.Username {
		return false, true
	}

	tx, err := txn.Start(ctx)
	if err != nil {
		b.log.Error(errors.WrapFail(err, "start txn"))
		return false, true
	}
	defer func() {
		err := tx.Close(ctx)
		if err != nil {
			b.log.Warn(errors.WrapFail(err, "close txn"))
		}
	}()

	scheduled, err := b.scheduleMeeting(ctx, candidate.Username, meet)
	if err != nil {
		b.log.Error(errors.WrapFail(err, "schedule meeting for candidate"))
		return false, true
	}
	if !scheduled {
		return false, false
	}

	scheduled, err = b.scheduleMeeting(ctx, interviewer.Username, meet)
	if err != nil {
		b.log.Error(errors.WrapFail(err, "schedule meet for interviewer"))
		return false, true
	}
	if !scheduled {
		return false, true
	}

	err = b.repo.Interviews().Schedule(ctx, iid, candidate, interviewer, meet)
	if err != nil {
		b.log.Error(errors.WrapFail(err, "schedule interview"))
		return false, true
	}

	err = tx.Commit(ctx)
	if err != nil {
		b.log.Error(errors.WrapFail(err, "commit txn"))
		return false, true
	}

	return true, true
}

func (b *Bot) scheduleMeeting(ctx context.Context, username string, meet models.Meeting) (bool, error) {
	user, err := b.repo.Users().Get(ctx, username)
	if err != nil {
		return false, errors.WrapFail(err, "find user")
	}

	if user == nil {
		return false, nil
	}

	insertIdx, can := user.AddMeeting(meet)
	if !can {
		return false, nil
	}
	meets := slices.Insert(user.Assigned, insertIdx, meet)

	assigned, err := b.repo.Users().UpdateMeetings(ctx, username, meets)
	if err != nil {
		return false, errors.WrapFail(err, "update meetings")
	}

	return assigned, nil
}