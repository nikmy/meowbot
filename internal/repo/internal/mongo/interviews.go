package repo

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/nikmy/meowbot/internal/repo/models"
	"github.com/nikmy/meowbot/pkg/errors"
	"github.com/nikmy/meowbot/pkg/mongotools"
)

type mongoInterviews struct {
	coll *mongo.Collection
}

func (m mongoInterviews) Create(ctx context.Context, vacancy string, candidateTg string) (id string, err error) {
	randomSuffix := strconv.Itoa(rand.Intn(90) + 10)
	timestamp := strconv.FormatInt(time.Now().UTC().UnixMicro(), 16)
	id = timestamp + randomSuffix

	r, err := m.coll.InsertOne(
		ctx,
		bson.M{
			"_id":                            id,
			models.InterviewFieldVacancy:     vacancy,
			models.InterviewFieldCandidate: candidateTg,
		},
	)
	if err != nil {
		return "", errors.WrapFail(err, "insert interview")
	}

	s, ok := r.InsertedID.(fmt.Stringer)
	if !ok {
		return "", errors.Error("cannot make string id")
	}

	return s.String(), nil
}

func (m mongoInterviews) Delete(ctx context.Context, id string) (bool, error) {
	r := m.coll.FindOneAndDelete(
		ctx,
		mongotools.FilterByID(id),
	)

	err := r.Err()
	if errors.Is(err, mongo.ErrNoDocuments) {
		return false, nil
	}
	if err != nil {
		return false, errors.WrapFail(err, "find one and delete")
	}
	return true, nil
}

func (m mongoInterviews) Schedule(ctx context.Context, id string, interviewerTg int64, slot models.Meeting) error {
	_, err := m.coll.UpdateOne(
		ctx,
		mongotools.FilterByID(id),
		bson.M{
			models.InterviewFieldInterviewerTg: interviewerTg,
			models.InterviewFieldInterval:      slot,
		},
	)
	return errors.WrapFail(err, "update interview")
}

func (m mongoInterviews) Notify(ctx context.Context, id string, at int64, notified models.Role) error {
	_, err := m.coll.UpdateOne(
		ctx,
		mongotools.FilterByID(id),
		bson.M{
			models.InterviewFieldLastNotification: bson.M{
				models.NotificationFieldUnixTime: at,
				models.NotificationFieldNotified: notified,
			},
		},
	)
	return errors.WrapFail(err, "update interview")
}

func (m mongoInterviews) Find(ctx context.Context, id string) (*models.Interview, error) {
	r := m.coll.FindOne(ctx, mongotools.FilterByID(id))
	if err := r.Err(); err != nil {
		return nil, errors.WrapFail(err, "find interview by id")
	}

	var parsed models.Interview
	err := r.Decode(&parsed)
	if err != nil {
		return nil, errors.WrapFail(err, "decode interview")
	}

	return &parsed, nil
}

func (m mongoInterviews) FindByUser(ctx context.Context, userTg int64) ([]models.Interview, error) {
	c, err := m.coll.Find(
		ctx,
		bson.M{"$or": []bson.M{
			{models.InterviewFieldCandidateTg: userTg},
			{models.InterviewFieldInterviewerTg: userTg},
		}})
	if err != nil {
		return nil, errors.WrapFail(err, "find interview")
	}

	parsed, err := mongotools.FilterFunc[models.Interview](ctx, c, nil)
	if err != nil {
		return nil, errors.WrapFail(err, "filter interviews")
	}

	return parsed, nil
}

func (m mongoInterviews) GetReadyAt(ctx context.Context, at int64) ([]models.Interview, error) {
	c, err := m.coll.Find(
		ctx,
		bson.M{"$and": bson.A{
			bson.M{models.InterviewFieldStatus: models.InterviewStatusScheduled},
			bson.M{models.InterviewFieldInterval: bson.M{"dim_cm.0": bson.A{"$lt", at}}},
			bson.M{"$or": bson.D{
				{models.InterviewFieldLastNotification, bson.A{"$exists", false}},
				{models.InterviewFieldLastNotification, bson.M{
					models.NotificationFieldUnixTime: bson.D{{"$gt", at - time.Minute.Milliseconds()}},
				}},
				{models.InterviewFieldLastNotification, bson.M{
					models.NotificationFieldNotified: bson.D{
						{"$or", bson.A{
							bson.D{{"$dim_cm.0", false}},
							bson.D{{"$dim_cm.1", false}},
						}},
					},
				}},
			}},
		}},
		&options.FindOptions{Max: 1024},
	)
	if err != nil {
		return nil, errors.WrapFail(err, "find interviews started at without recent notifications")
	}

	ready, err := mongotools.FilterFunc[models.Interview](ctx, c, nil)
	return ready, errors.WrapFail(err, "parse interviews")
}

func (m mongoInterviews) Cancel(ctx context.Context, id string, side models.Role) error {
	r, err := m.coll.UpdateOne(
		ctx,
		mongotools.FilterByID(id),
		bson.M{
			models.InterviewFieldStatus:      models.InterviewStatusCancelled,
			models.InterviewFieldCancelledBy: side,
			models.InterviewFieldInterval:    [2]int{},
			models.InterviewFieldLastNotification: nil,
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

func (m mongoInterviews) Done(ctx context.Context, id string) (err error) {
	r, err := m.coll.UpdateOne(
		ctx,
		mongotools.FilterByID(id),
		bson.M{models.InterviewFieldStatus: models.InterviewStatusFinished},
	)
	if err != nil {
		return errors.WrapFail(err, "update interview by id")
	}

	if r.ModifiedCount == 0 {
		return errors.Error("no interviews updated")
	}

	return nil
}
