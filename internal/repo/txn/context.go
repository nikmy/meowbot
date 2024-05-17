package txn

import (
	"context"

	"github.com/nikmy/meowbot/pkg/errors"
	"github.com/nikmy/meowbot/pkg/logger"
)

func getFromCtx(ctx context.Context) (Txn, logger.Logger) {
	return ctx.Value("txn").(Txn), ctx.Value("txnLog").(logger.Logger)
}

func Start(ctx context.Context) bool {
	txn, log := getFromCtx(ctx)
	err := txn.Start(ctx)
	if err != nil {
		log.Error(errors.Wrap(err, "start transaction"))
		return false
	}

	return true
}

func Abort(ctx context.Context) {
	txn, log := getFromCtx(ctx)
	log.Error(errors.WrapFail(txn.Abort(ctx), "abort transaction"))
}

func Commit(ctx context.Context) {
	txn, log := getFromCtx(ctx)
	log.Error(errors.WrapFail(txn.Commit(ctx), "commit transaction"))
}
