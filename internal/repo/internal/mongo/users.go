package repo

import (
	"context"

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

func (u mongoUsers) Update(
	ctx context.Context,
	username string,
	telegramID *int64,
	category *models.UserCategory,
	intGrade *int,
) (*models.User, error) {
	r := u.coll.FindOneAndUpdate(
		ctx,
		mongotools.Field(models.UserFieldUsername, &username),
		mongotools.SetAll(
			mongotools.Field(models.UserFieldTelegram, telegramID),
			mongotools.Field(models.UserFieldCategory, category),
			mongotools.Field(models.UserFieldIntGrade, &intGrade),
		),
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

func (u mongoUsers) UpdateMeetings(ctx context.Context, username string, meets []models.Meeting) (bool, error) {
	r, err := u.coll.UpdateOne(
		ctx,
		bson.M{models.UserFieldUsername: username},
		bson.M{"$set": bson.M{models.UserFieldAssigned: meets}},
	)
	if err != nil {
		return false, errors.WrapFail(err, "update user")
	}

	return r.ModifiedCount == 1, nil
}

func interviewersOnly() bson.M {
	return bson.M{models.UserFieldIntGrade: bson.D{{"$gt", models.GradeNotInterviewer}}}
}
