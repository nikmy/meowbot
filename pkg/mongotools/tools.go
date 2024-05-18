package mongotools

import (
	"context"
	"strconv"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/nikmy/meowbot/pkg/errors"
)

func SetAll(fieldKVs ...bson.M) bson.M {
	s := make(map[string]any, len(fieldKVs))
	for _, kv := range fieldKVs {
		for k, v := range kv {
			s[k] = v
		}
	}

	return bson.M{"$set": bson.M(s)}
}

func All() bson.M {
	return bson.M{}
}

func ID(id string) bson.M {
	return bson.M{"_id": id}
}

func Path(keys ...string) string {
	return strings.Join(keys, ".")
}

func Index(path string, i int) string {
	return path+"."+strconv.Itoa(i)
}

func Field[T any](field string, value *T) bson.M {
	if value == nil {
		return nil
	}
	return bson.M{field: value}
}

func AtMost[T any](ctx context.Context, c *mongo.Cursor, limit int) ([]T, error) {
	switch {
	case limit == 0:
		_ = c.Close(ctx)
		return nil, nil
	case limit < 0:
		return FilterFunc[T](ctx, c, nil)
	}

	defer c.Close(ctx)

	parsed := make([]T, 0, limit)
	for c.Next(ctx) {
		var item T
		err := c.Decode(&item)
		if err != nil {
			return nil, errors.WrapFail(err, "decode item")
		}

		parsed = append(parsed, item)
		if len(parsed) == limit {
			break
		}
	}

	return parsed[:len(parsed):len(parsed)], c.Err()
}

func FilterFunc[T any](ctx context.Context, c *mongo.Cursor, filterFunc func(T) bool) ([]T, error) {
	defer c.Close(ctx)

	var filtered []T
	for c.Next(ctx) {
		var item T
		err := c.Decode(&item)
		if err != nil {
			return nil, errors.WrapFail(err, "decode item")
		}

		if filterFunc == nil || filterFunc(item) {
			filtered = append(filtered, item)
		}
	}

	return filtered, c.Err()
}
