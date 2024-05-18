package txn

import (
	"context"
	"time"

	"github.com/nikmy/meowbot/pkg/errors"
	"github.com/nikmy/meowbot/pkg/logger"
)

type Session interface {
	Txn() Txn
	Close(ctx context.Context)
}

type sessionManager interface {
	NewSession() (Session, error)
}

func NewManager(ctx context.Context, l logger.Logger, m sessionManager, txnTimeout time.Duration) Manager {
	return Manager{
		ctx:     ctx,
		log:     l,
		manager: m,
		maxTime: txnTimeout,
	}
}

type Manager struct {
	ctx     context.Context
	log     logger.Logger
	manager sessionManager
	maxTime time.Duration
}

func (m Manager) NewSessionContext(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc, error) {
	session, err := m.manager.NewSession()
	if err != nil {
		return nil, nil, errors.WrapFail(err, "create session context")
	}

	ctx, cancel := context.WithTimeout(parent, timeout)
	ctx = context.WithValue(ctx, "txn_session", session)
	context.AfterFunc(ctx, func() {
		closeCtx, cancelClose := context.WithTimeout(context.Background(), time.Second)
		defer cancelClose()
		session.Close(closeCtx)
	})

	return ctx, cancel, nil
}

func Start(ctx context.Context) (Txn, error) {
	tx := ctx.Value("txn_session").(Session).Txn()
	return tx, tx.Start(ctx)
}
