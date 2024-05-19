package repo

import (
	"context"

	"github.com/chenmingyong0423/go-mongox"
	"github.com/chenmingyong0423/go-mongox/builder/query"
	"github.com/chenmingyong0423/go-mongox/builder/update"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/nikmy/meowbot/internal/repo/models"
	"github.com/nikmy/meowbot/pkg/errors"
	mng "github.com/nikmy/meowbot/pkg/mongotools"
)

type mongoUsers struct {
	c *mongox.Collection[models.User]
}

func (u mongoUsers) Update(
	ctx context.Context,
	username string,
	telegramID *int64,
	category *models.UserCategory,
	intGrade *int,
) (*models.User, error) {
	return u.findOneAndUpdate(ctx, username, telegramID, category, intGrade, false)
}

func (u mongoUsers) Upsert(
	ctx context.Context,
	username string,
	telegramID *int64,
	category *models.UserCategory,
	intGrade *int,
) (*models.User, error) {
	return u.findOneAndUpdate(ctx, username, telegramID, category, intGrade, true)
}

func (u mongoUsers) findOneAndUpdate(
	ctx context.Context,
	username string,
	telegramID *int64,
	category *models.UserCategory,
	intGrade *int,
	upsert bool,
) (*models.User, error) {
	upd := update.BsonBuilder()
	if telegramID != nil {
		upd.Set(models.UserFieldTelegram, *telegramID)
	}
	if category != nil {
		upd.Set(models.UserFieldCategory, *category)
	}
	if intGrade != nil {
		upd.Set(models.UserFieldIntGrade, *intGrade)
	}

	r := u.c.Collection().FindOneAndUpdate(
		ctx,
		mng.Field(models.UserFieldUsername, &username),
		upd.Build(),
		options.FindOneAndUpdate().SetUpsert(upsert),
	)

	err := r.Err()
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}

	if err != nil {
		return nil, errors.WrapFail(err, "do findOneAndUpdate")
	}

	var parsed models.User
	err = r.Decode(&parsed)
	if err != nil {
		return nil, errors.WrapFail(err, "parse user")
	}

	return &parsed, nil
}

func (u mongoUsers) Get(ctx context.Context, username string) (*models.User, error) {
	r := u.c.Collection().FindOne(ctx, mng.Field(models.UserFieldUsername, &username))
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
	interviewersOnly := query.Gt(models.UserFieldIntGrade, models.GradeNotInterviewer)

	c, err := u.c.Collection().Find(ctx, interviewersOnly)
	if err != nil {
		return nil, errors.WrapFail(err, "select users to match")
	}

	matched, err := mng.FilterFunc(ctx, c, func(user models.User) bool {
		_, canAdd := user.AddMeeting(slot)
		return canAdd
	})
	if err != nil {
		return nil, errors.WrapFail(err, "filter users")
	}

	return matched, nil
}

func (u mongoUsers) UpdateMeetings(ctx context.Context, username string, meets []models.Meeting) (bool, error) {
	r, err := u.c.Updater().
		Filter(query.Eq(models.UserFieldUsername, username)).
		Updates(update.Set(models.UserFieldAssigned, meets)).
		UpdateOne(ctx)

	if err != nil {
		return false, errors.WrapFail(err, "update user")
	}

	return r.ModifiedCount == 1, nil
}
