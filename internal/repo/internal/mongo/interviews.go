package repo

import (
	"context"
	"math/rand"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/nikmy/meowbot/internal/repo/models"
	"github.com/nikmy/meowbot/pkg/errors"
	mng "github.com/nikmy/meowbot/pkg/mongotools"
)

type mongoInterviews struct {
	coll *mongo.Collection
}

func (m mongoInterviews) Create(ctx context.Context, vacancy string, candidateTg string) (string, error) {
	randomSuffix := strconv.Itoa(rand.Intn(90) + 10)
	timestamp := strconv.FormatInt(time.Now().UnixMicro(), 16)
	id := timestamp + randomSuffix

	_, err := m.coll.InsertOne(
		ctx,
		bson.M{
			"_id":                            id,
			models.InterviewFieldVacancy:     vacancy,
			models.InterviewFieldCandidateUN: candidateTg,
		},
	)
	if err != nil {
		return "", errors.WrapFail(err, "insert interview")
	}

	return id, nil
}

func (m mongoInterviews) Delete(ctx context.Context, id string) (*models.Interview, error) {
	r := m.coll.FindOneAndDelete(ctx, mng.ID(id))
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
	slot models.Meeting,
) error {
	_, err := m.coll.UpdateOne(
		ctx,
		mng.ID(id),
		bson.M{"$set": bson.M{
			models.InterviewFieldStatus:        models.InterviewStatusScheduled,
			models.InterviewFieldInterviewerTg: interviewer.Telegram,
			models.InterviewFieldCandidateTg:   candidate.Telegram,
			models.InterviewFieldInterviewerUN: interviewer.Username,
			models.InterviewFieldMeet:          slot,
		}},
	)
	return errors.WrapFail(err, "update interview")
}

func (m mongoInterviews) Notify(ctx context.Context, id string, at int64, notified [2]bool) error {
	notifiedField := mng.Path(models.InterviewFieldLastNotification, models.NotificationFieldNotified)
	unixTimeField := mng.Path(models.InterviewFieldLastNotification, models.NotificationFieldUnixTime)

	_, err := m.coll.UpdateOne(
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
	r := m.coll.FindOne(ctx, mng.ID(id))
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
	c, err := m.coll.Find(
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

	_, err := m.coll.UpdateOne(ctx, mng.ID(id), update)
	return errors.WrapFail(err, "update one interview")
}

func (m mongoInterviews) GetStartedWithin(ctx context.Context, from, to int64) ([]models.Interview, error) {
	unixTime := mng.Path(models.InterviewFieldLastNotification, models.NotificationFieldUnixTime)
	notified := mng.Path(models.InterviewFieldLastNotification, models.NotificationFieldNotified)

	query := bson.M{"$and": bson.A{
		bson.M{
			models.InterviewFieldStatus:             models.InterviewStatusScheduled,
			mng.Index(models.InterviewFieldMeet, 0): bson.M{"$lt": to},
		},
		bson.M{"$or": bson.A{
			bson.M{models.InterviewFieldLastNotification: bson.M{"$exists": false}},
			bson.M{unixTime: bson.M{"$gt": from}},
			bson.M{mng.Index(notified, int(models.RoleInterviewer)): false},
			bson.M{mng.Index(notified, int(models.RoleCandidate)): false},
		}},
	}}

	c, err := m.coll.Find(ctx, query, new(options.FindOptions).SetLimit(1024))
	if err != nil {
		return nil, errors.WrapFail(err, "find interviews started at without recent notifications")
	}

	ready, err := mng.FilterFunc[models.Interview](ctx, c, nil)
	return ready, errors.WrapFail(err, "parse interviews")
}

func (m mongoInterviews) Cancel(ctx context.Context, id string, side models.Role) error {
	r, err := m.coll.UpdateOne(
		ctx,
		mng.ID(id),
		bson.M{
			"$unset": bson.M{
				models.InterviewFieldMeet:             true,
				models.InterviewFieldLastNotification: true,
				models.InterviewFieldInterviewerTg:    true,
				models.InterviewFieldInterviewerUN:    true,
				models.InterviewFieldZoom:             true,
			},
			"$set": bson.M{
				models.InterviewFieldStatus:      models.InterviewStatusCancelled,
				models.InterviewFieldCancelledBy: side,
			},
		},
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
	r, err := m.coll.UpdateOne(
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
	_, err := m.coll.UpdateMany(
		ctx,
		mng.Field(models.InterviewFieldCandidateUN, &username),
		mng.SetAll(mng.Field(models.InterviewFieldCandidateTg, &tg)),
	)
	return errors.WrapFail(err, "update many documents")
}
