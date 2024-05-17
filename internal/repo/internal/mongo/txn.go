package repo

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"

	"github.com/nikmy/meowbot/internal/repo/txn"
	"github.com/nikmy/meowbot/pkg/errors"
)

func (m *mongoClient) Make(lvl txn.Model) txn.Txn {
	return &mongoTxn{lvl: lvl, c: m.c}
}

type mongoTxn struct {
	lvl txn.Model
	c *mongo.Client
	s mongo.Session
	sc       mongo.SessionContext
	finished bool
}

func (m *mongoTxn) Close(ctx context.Context) error {
	defer m.s.EndSession(ctx)
	if !m.finished {
		return m.Abort(ctx)
	}

	return nil
}

func (m *mongoTxn) Start(ctx context.Context) error {
	var err error

	enabled := true
	switch m.lvl {
	case txn.ModelSnapshotIsolation:
		m.s, err = m.c.StartSession(&options.SessionOptions{Snapshot: &enabled})
	case txn.ModelSerializable:
		m.s, err = m.c.StartSession(&options.SessionOptions{CausalConsistency: &enabled})
	case txn.ModelStrictSerializable:
		m.s, err = m.c.StartSession(&options.SessionOptions{
			DefaultReadConcern: readconcern.Linearizable(),
			DefaultWriteConcern: writeconcern.Majority(),
			CausalConsistency: &enabled,
		})
	default:
		return errors.Error("unsupported consistency level \"%d\"", m.lvl)
	}

	if err != nil {
		return errors.WrapFail(err, "start mongo session")
	}

	m.sc = mongo.NewSessionContext(ctx, m.s)
	return m.sc.StartTransaction()
}

func (m *mongoTxn) Abort(ctx context.Context) error {
	defer m.s.EndSession(ctx)
	m.finished = true
	return m.sc.AbortTransaction(ctx)
}

func (m *mongoTxn) Commit(ctx context.Context) error {
	defer m.s.EndSession(ctx)
	m.finished = true
	return m.sc.CommitTransaction(ctx)
}
