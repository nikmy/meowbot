package txn

import (
	"context"

	"github.com/nikmy/meowbot/pkg/errors"
	"github.com/nikmy/meowbot/pkg/logger"
)

type maker interface {
	Make(lvl Model) Txn
}

func NewManager(l logger.Logger, m maker) Manager {
	return Manager{
		log:   l,
		maker: m,
	}
}

type Manager struct {
	ctx   context.Context
	log   logger.Logger
	maker maker
}

func (m Manager) NewContext(parent context.Context, lvl Model) (context.Context, context.CancelFunc) {
	ctx := context.WithValue(parent, "txnLog", m.log)

	txn := m.maker.Make(lvl)
	ctx = context.WithValue(parent, "txn", txn)
	context.AfterFunc(ctx, func() {
		m.log.Error(errors.WrapFail(txn.Close(m.ctx), "close transaction"))
	})

	return context.WithCancel(ctx)
}
