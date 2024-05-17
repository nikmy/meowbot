package repo

import (
	"context"
	"slices"
	"sort"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/nikmy/meowbot/internal/repo/models"
	"github.com/nikmy/meowbot/pkg/errors"
	"github.com/nikmy/meowbot/pkg/mongotools"
)

type mongoUsers struct {
	coll *mongo.Collection
}

func (u mongoUsers) Upsert(
	ctx context.Context,
	username string,
	telegramID *int64,
	category *models.UserCategory,
	intGrade *int,
) (*models.User, error) {
	upsert := true
	r := u.coll.FindOneAndUpdate(
		ctx,
		mongotools.Field(models.UserFieldUsername, &username),
		mongotools.SetAll(
			mongotools.Field(models.UserFieldTelegram, telegramID),
			mongotools.Field(models.UserFieldCategory, category),
			mongotools.Field(models.UserFieldIntGrade, &intGrade),
		),
		&options.FindOneAndUpdateOptions{Upsert: &upsert},
	)

	err := r.Err()
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}

	if err != nil {
		return nil, errors.WrapFail(err, "do upsert")
	}

	var parsed models.User
	err = r.Decode(&parsed)
	if err != nil {
		return nil, errors.WrapFail(err, "parse user")
	}

	return &parsed, nil
}

func (u mongoUsers) Get(ctx context.Context, username string) (*models.User, error) {
	r := u.coll.FindOne(ctx, mongotools.Field(models.UserFieldUsername, &username))
	if r.Err() != nil {
		return nil, errors.WrapFail(r.Err(), "find user by username")
	}

	var user models.User
	err := r.Decode(&user)
	if err != nil {
		return nil, errors.WrapFail(err, "decode user")
	}

	return &user, nil
}

func (u mongoUsers) Match(ctx context.Context, slot [2]int64) ([]models.User, error) {
	c, err := u.coll.Find(ctx, interviewersOnly())
	if err != nil {
		return nil, errors.WrapFail(err, "select users to match")
	}

	matched, err := mongotools.FilterFunc(ctx, c, func(user models.User) bool {
		_, canAdd := user.AddMeeting(slot)
		return canAdd
	})
	if err != nil {
		return nil, errors.WrapFail(err, "filter users")
	}

	return matched, nil
}

func (u mongoUsers) Schedule(ctx context.Context, username string, meeting models.Meeting) (bool, error) {
	user, err := u.Get(ctx, username)
	if err != nil {
		return false, errors.WrapFail(err, "find user")
	}

	if user.IntGrade == models.GradeNotInterviewer {
		return false, nil
	}

	insertIdx, can := user.AddMeeting(meeting)
	if !can {
		return false, nil
	}

	user.Meetings = slices.Insert(user.Meetings, insertIdx, meeting)
	r, err := u.coll.UpdateOne(
		ctx,
		bson.M{models.UserFieldUsername: username},
		bson.M{models.UserFieldMeetings: user.Meetings},
	)
	if err != nil {
		return false, errors.WrapFail(err, "update user")
	}

	return r.ModifiedCount == 1, nil
}

func (u mongoUsers) Free(ctx context.Context, username string, meeting models.Meeting) error {
	user, err := u.Get(ctx, username)
	if err != nil {
		return errors.WrapFail(err, "find user")
	}

	idx := sort.Search(len(user.Meetings), func(i int) bool {
		return user.Meetings[i][0] == meeting[0]
	})
	if idx == len(user.Meetings) {
		return errors.Error("no meetings with specified start")
	}

	if user.Meetings[idx][1] != meeting[1] {
		return errors.Error("no meetings with specified end")
	}

	user.Meetings = slices.Delete(user.Meetings, idx, idx+1)
	r, err := u.coll.UpdateOne(
		ctx,
		bson.M{models.UserFieldUsername: username},
		bson.M{models.UserFieldMeetings: user.Meetings},
	)
	if err != nil {
		return errors.WrapFail(err, "update user")
	}

	if r.ModifiedCount == 0 {
		return errors.Error("no users modified")
	}

	return nil
}

func interviewersOnly() bson.M {
	return bson.M{models.UserFieldIntGrade: bson.D{{"$gt", models.GradeNotInterviewer}}}
}
