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
		ID: id,
		Vacancy: vacancy,
		CandidateUN: candidate,
	})
	if err != nil {
		return "", errors.WrapFail(err, "insert interview")
	}

	return id, nil
}

func (m mongoInterviews) Delete(ctx context.Context, id string) (*models.Interview, error) {
	r := m.c.Collection().FindOneAndDelete(ctx, mng.ID(id))
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
	_, err := m.c.Collection().UpdateOne(
		ctx,
		mng.ID(id),
		update.BsonBuilder().
			Set(models.InterviewFieldStatus, models.InterviewStatusScheduled).
			Set(models.InterviewFieldInterviewerTg, interviewer.Telegram).
			Set(models.InterviewFieldInterviewerUN, interviewer.Username).
			Set(models.InterviewFieldCandidateTg, candidate.Telegram).
			Set(models.InterviewFieldMeet, meet).
			Build(),
	)
	return errors.WrapFail(err, "update interview")
}

func (m mongoInterviews) Notify(ctx context.Context, id string, at int64, notified [2]bool) error {
	notifiedField := mng.Path(models.InterviewFieldLastNotification, models.NotificationFieldNotified)
	unixTimeField := mng.Path(models.InterviewFieldLastNotification, models.NotificationFieldUnixTime)

	_, err := m.c.Collection().UpdateOne(
		ctx,
		mng.ID(id),
		bson.M{"$set": bson.M{
			unixTimeField: at,
			notifiedField: notified,
		}},
	)
	return errors.WrapFail(err, "update interview")
}

func (m mongoInterviews) Find(ctx context.Context, id string) (*models.Interview, error) {
	r := m.c.Collection().FindOne(ctx, mng.ID(id))
	err := r.Err()

	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}

	if err != nil {
		return nil, errors.WrapFail(err, "find interview by id")
	}

	var parsed models.Interview
	err = r.Decode(&parsed)
	if err != nil {
		return nil, errors.WrapFail(err, "decode interview")
	}

	return &parsed, nil
}

func (m mongoInterviews) FindByUser(ctx context.Context, username string) ([]models.Interview, error) {
	c, err := m.c.Collection().Find(
		ctx,
		bson.M{"$or": []bson.M{
			{models.InterviewFieldCandidateUN: username},
			{models.InterviewFieldInterviewerUN: username},
		}})
	if err != nil {
		return nil, errors.WrapFail(err, "find interview")
	}

	parsed, err := mng.FilterFunc[models.Interview](ctx, c, nil)
	if err != nil {
		return nil, errors.WrapFail(err, "filter interviews")
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
	update := mng.SetAll(
		mng.Field(models.InterviewFieldVacancy, vacancy),
		mng.Field(models.InterviewFieldCandidateUN, candidate),
		mng.Field(models.InterviewFieldData, data),
		mng.Field(models.InterviewFieldZoom, zoom),
	)

	if candidate != nil {
		update["$unset"] = bson.M{models.InterviewFieldCandidateTg: ""}
	}

	_, err := m.c.Collection().UpdateOne(ctx, mng.ID(id), update)
	return errors.WrapFail(err, "update one interview")
}

func (m mongoInterviews) GetUpcoming(ctx context.Context, lastNotifyBefore, startsBefore int64) ([]models.Interview, error) {
	unixTime := mng.Path(models.InterviewFieldLastNotification, models.NotificationFieldUnixTime)
	notified := mng.Path(models.InterviewFieldLastNotification, models.NotificationFieldNotified)

	q := query.BsonBuilder().
		And(
			query.Eq(models.InterviewFieldStatus, models.InterviewStatusScheduled),
			bson.D{{mng.Index(models.InterviewFieldMeet, 0), bson.M{"$lt": startsBefore}}},
			query.Or(
				query.Exists(models.InterviewFieldLastNotification, true),
				query.Lt(unixTime, lastNotifyBefore),
				bson.D{
					{mng.Index(notified, int(models.RoleInterviewer)), false},
					{mng.Index(notified, int(models.RoleCandidate)), false},
				},
			),
		).
		Build()

	c, err := m.c.Collection().Find(ctx, q)
	if err != nil {
		return nil, errors.WrapFail(err, "find interviews started at without recent notifications")
	}

	ready, err := mng.AtMost[models.Interview](ctx, c, 1024)
	return ready, errors.WrapFail(err, "parse interviews")
}

func (m mongoInterviews) Cancel(ctx context.Context, id string, side models.Role) error {
	r, err := m.c.Collection().UpdateOne(
		ctx,
		query.Id(id),
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
	)
	if err != nil {
		return errors.WrapFail(err, "update interview by id")
	}

	if r.ModifiedCount == 0 {
		return errors.Error("no interviews updated")
	}

	return nil
}

func (m mongoInterviews) Done(ctx context.Context, id string) error {
	r, err := m.c.Collection().UpdateOne(
		ctx,
		mng.ID(id),
		bson.M{"$set": bson.M{models.InterviewFieldStatus: models.InterviewStatusFinished}},
	)
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
