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

func (m Manager) NewContext(parent context.Context, lvl Model, do func() error) bool {
	ctx := context.WithValue(parent, "txnLog", m.log)

	txn := m.maker.Make(lvl)
	ctx = context.WithValue(parent, "txn", txn)
	context.AfterFunc(ctx, func() {
		m.log.Error(errors.WrapFail(txn.Close(m.ctx), "close transaction"))
	})

	err := txn.Start(parent)
	if err != nil {
		m.log.Error(errors.Wrap(err, "start transaction"))
		return false
	}

	err = do()

	if err != nil {
		m.log.Info(errors.Wrap(err, "aborting transaction"))
		err = txn.Abort(parent)
		if err != nil {
			m.log.Error(errors.WrapFail(err, "abort transaction"))
		}
		return false
	}

	err = txn.Commit(parent)
	if err != nil {
		m.log.Error(errors.WrapFail(err, "commit transaction"))
		return false
	}

	return true
}
