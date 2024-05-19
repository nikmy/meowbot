package mongotools

import (
	"context"
	"strconv"
	"strings"

	"go.mongodb.org/mongo-driver/mongo"

	"github.com/nikmy/meowbot/pkg/errors"
)

func Path(keys ...string) string {
	return strings.Join(keys, ".")
}

func Index(path string, i int) string {
	return path + "." + strconv.Itoa(i)
}

func FilterFunc[T any](ctx context.Context, c *mongo.Cursor, limit *int, filterFunc func(T) bool) ([]T, error) {
	defer c.Close(ctx)

	if limit != nil && *limit == 0 {
		return nil, nil
	}

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

		if limit != nil && *limit > 0 && len(filtered) == *limit {
			break
		}
	}

	return filtered, c.Err()
}
