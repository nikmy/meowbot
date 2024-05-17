package txn

import (
	"context"
	"time"

	"github.com/nikmy/meowbot/pkg/errors"
	"github.com/nikmy/meowbot/pkg/logger"
)

type maker interface {
	Make(lvl Model) Txn
}

func NewManager(ctx context.Context, l logger.Logger, m maker, txnTimeout time.Duration) Manager {
	return Manager{
		ctx: ctx,
		log:   l,
		maker: m,
		maxTime: txnTimeout,
	}
}

type Manager struct {
	ctx   context.Context
	log   logger.Logger
	maker maker
	maxTime time.Duration
}

func (m Manager) NewContext(parent context.Context, lvl Model) (context.Context, context.CancelFunc) {
	ctx := context.WithValue(parent, "txnLog", m.log)

	txn := m.maker.Make(lvl)
	ctx = context.WithValue(parent, "txn", txn)
	context.AfterFunc(ctx, func() {
		m.log.Error(errors.WrapFail(txn.Close(m.ctx), "close transaction"))
	})

	return context.WithTimeout(ctx, time.Second*5)
}
