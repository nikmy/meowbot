package repo

import (
	"context"
	"math/rand"
	"strconv"
	"time"

	"github.com/chenmingyong0423/go-mongox"
	"github.com/chenmingyong0423/go-mongox/builder/query"
	"github.com/chenmingyong0423/go-mongox/builder/update"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/nikmy/meowbot/internal/repo/models"
	"github.com/nikmy/meowbot/pkg/errors"
	mng "github.com/nikmy/meowbot/pkg/mongotools"
)

type mongoInterviews struct {
	c *mongox.Collection[models.Interview]
}

func (m mongoInterviews) Create(ctx context.Context, vacancy string, candidate string) (string, error) {
	randomSuffix := strconv.Itoa(rand.Intn(90) + 10)
	timestamp := strconv.FormatInt(time.Now().UnixMicro(), 16)
	id := timestamp + randomSuffix

	_, err := m.c.Creator().InsertOne(ctx, &models.Interview{
		ID:          id,
		Vacancy:     vacancy,
		CandidateUN: candidate,
	})
	if err != nil {
		return "", errors.WrapFail(err, "insert interview")
	}

	return id, nil
}

func (m mongoInterviews) Delete(ctx context.Context, id string) (*models.Interview, error) {
	r := m.c.Collection().FindOneAndDelete(ctx, query.Id(id))
	err := r.Err()

	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}

	if err != nil {
		return nil, errors.WrapFail(err, "find one and delete")
	}

	var parsed models.Interview
	err = r.Decode(&parsed)
	if err != nil {
		return nil, errors.WrapFail(err, "decode deleted interview")
	}

	return &parsed, nil
}

func (m mongoInterviews) Schedule(
	ctx context.Context,
	id string,
	candidate models.User,
	interviewer models.User,
	meet models.Meeting,
) error {
	_, err := m.c.Updater().
		Filter(query.Id(id)).
		Updates(update.BsonBuilder().
			Set(models.InterviewFieldStatus, models.InterviewStatusScheduled).
			Set(models.InterviewFieldInterviewerTg, interviewer.Telegram).
			Set(models.InterviewFieldInterviewerUN, interviewer.Username).
			Set(models.InterviewFieldCandidateTg, candidate.Telegram).
			Set(models.InterviewFieldMeet, meet).
			Build()).
		UpdateOne(ctx)
	return errors.WrapFail(err, "update interview")
}

func (m mongoInterviews) Notify(ctx context.Context, id string, at int64, notified [2]bool) error {
	notifiedField := mng.Path(models.InterviewFieldLastNotification, models.NotificationFieldNotified)
	unixTimeField := mng.Path(models.InterviewFieldLastNotification, models.NotificationFieldUnixTime)

	_, err := m.c.Updater().
		Filter(query.Id(id)).
		Updates(
			update.BsonBuilder().
				Set(unixTimeField, at).
				Set(notifiedField, notified).
				Build(),
		).
		UpdateOne(ctx)
	return errors.WrapFail(err, "update interview")
}

func (m mongoInterviews) Find(ctx context.Context, id string) (*models.Interview, error) {
	r, err := m.c.Finder().
		Filter(query.Id(id)).
		FindOne(ctx)

	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}

	if err != nil {
		return nil, errors.WrapFail(err, "find interview by id")
	}

	return r, nil
}

func (m mongoInterviews) FindByUser(ctx context.Context, username string) ([]*models.Interview, error) {
	parsed, err := m.c.Finder().
		Filter(query.Or(
			query.Eq(models.InterviewFieldCandidateUN, username),
			query.Eq(models.InterviewFieldInterviewerUN, username),
		)).
		Find(ctx)

	if err != nil {
		return nil, errors.WrapFail(err, "find interview")
	}

	return parsed, nil
}

func (m mongoInterviews) Update(
	ctx context.Context,
	id string,
	vacancy *string,
	candidate *string,
	data *[]byte,
	zoom *string,
) error {
	upd := update.BsonBuilder().
		Set(models.InterviewFieldVacancy, vacancy).
		Set(models.InterviewFieldCandidateUN, candidate).
		Set(models.InterviewFieldData, data).
		Set(models.InterviewFieldZoom, zoom)
	if candidate != nil {
		upd.Unset(models.InterviewFieldCandidateTg)
	}

	_, err := m.c.Updater().
		Filter(query.Id(id)).
		Updates(upd.Build()).
		UpdateOne(ctx)
	return errors.WrapFail(err, "update one interview")
}

func (m mongoInterviews) GetUpcoming(ctx context.Context, lastNotifyBefore, startsBefore int64) ([]*models.Interview, error) {
	unixTime := mng.Path(models.InterviewFieldLastNotification, models.NotificationFieldUnixTime)
	notified := mng.Path(models.InterviewFieldLastNotification, models.NotificationFieldNotified)

	q := query.BsonBuilder().
		And(
			query.Eq(models.InterviewFieldStatus, models.InterviewStatusScheduled),
			bson.D{{mng.Index(models.InterviewFieldMeet, 0), bson.M{"$lt": startsBefore}}},
			query.Or(
				query.Exists(models.InterviewFieldLastNotification, false),
				query.Lt(unixTime, lastNotifyBefore),
				query.Eq(mng.Index(notified, int(models.RoleInterviewer)), false),
				query.Eq(mng.Index(notified, int(models.RoleCandidate)), false),
			),
		).
		Build()

	ready, err := m.c.Finder().
		Filter(q).
		Find(ctx, options.Find().SetLimit(1024))

	return ready, errors.WrapFail(err, "find interviews started at without recent notifications")
}

func (m mongoInterviews) Cancel(ctx context.Context, id string, side models.Role) error {
	r, err := m.c.Updater().
		Filter(query.Id(id)).
		Updates(
			update.BsonBuilder().
				Unset(
					models.InterviewFieldMeet,
					models.InterviewFieldLastNotification,
					models.InterviewFieldInterviewerTg,
					models.InterviewFieldInterviewerUN,
					models.InterviewFieldZoom,
				).
				Set(models.InterviewFieldStatus, models.InterviewStatusCancelled).
				Set(models.InterviewFieldCancelledBy, side).
				Build(),
		).
		UpdateOne(ctx)
	if err != nil {
		return errors.WrapFail(err, "update interview by id")
	}

	if r.ModifiedCount == 0 {
		return errors.Error("no interviews updated")
	}

	return nil
}

func (m mongoInterviews) Done(ctx context.Context, id string) error {
	r, err := m.c.Updater().
		Filter(query.Id(id)).
		Updates(
			update.Set(models.InterviewFieldStatus, models.InterviewStatusFinished),
		).UpdateOne(ctx)
	if err != nil {
		return errors.WrapFail(err, "update interview by id")
	}

	if r.ModifiedCount == 0 {
		return errors.Error("no interviews updated")
	}

	return nil
}

func (m mongoInterviews) FixTg(ctx context.Context, username string, tg int64) error {
	_, err := m.c.Updater().
		Filter(query.Eq(models.InterviewFieldCandidateUN, username)).
		Updates(update.Set(models.InterviewFieldCandidateTg, tg)).
		UpdateMany(ctx)
	if err != nil {
		return errors.WrapFail(err, "fix tg for candidate")
	}

	_, err = m.c.Updater().
		Filter(query.Eq(models.InterviewFieldInterviewerUN, username)).
		Updates(update.Set(models.InterviewFieldInterviewerTg, tg)).
		UpdateMany(ctx)
	if err != nil {
		return errors.WrapFail(err, "fix tg for interviewer")
	}

	return nil
}
