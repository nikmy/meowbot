package txn

import (
	"context"
	"time"

	"github.com/nikmy/meowbot/pkg/errors"
)

type Session interface {
	Txn() Txn
	TxnWithModel(model Model) Txn
	Close(ctx context.Context)
}

type sessionManager interface {
	NewSession() (Session, error)
}

func NewManager(m sessionManager) Manager {
	return Manager{m}
}

type Manager struct {
	sessionManager
}

func (m Manager) NewSessionContext(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc, error) {
	session, err := m.NewSession()
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
