package txn

import (
	"context"
	"time"

	"github.com/nikmy/meowbot/pkg/errors"
)

type sessionManager interface {
	NewSession() (Session, error)
}

func NewManager(m sessionManager) Manager {
	return Manager{m}
}

type Manager struct {
	sessionManager
}

type sessionKey struct{}

func (m Manager) NewSessionContext(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc, error) {
	session, err := m.NewSession()
	if err != nil {
		return nil, nil, errors.WrapFail(err, "create session context")
	}

	ctx, cancel := context.WithTimeout(parent, timeout)
	context.AfterFunc(ctx, func() {
		closeCtx, cancelClose := context.WithTimeout(context.Background(), time.Second)
		defer cancelClose()
		session.Close(closeCtx)
	})

	ctx = context.WithValue(parent, sessionKey{}, session)
	ctx = session.BindContext(ctx)

	return ctx, cancel, nil
}

func Start(ctx context.Context, c Consistency, i Isolation) (ActiveTxn, error) {
	session, ok := ctx.Value(sessionKey{}).(Session)
	if !ok {
		return nil, errors.Fail("get session from context")
	}

	tx := session.Txn()
	return tx, errors.WrapFail(tx.Start(ctx), "start txn")
}
