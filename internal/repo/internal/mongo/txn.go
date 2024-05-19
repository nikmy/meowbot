package repo

import (
	"context"
	"sync/atomic"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"

	"github.com/nikmy/meowbot/pkg/errors"
	"github.com/nikmy/meowbot/pkg/txn"
)

func (m *mongoClient) NewSession() (txn.Session, error) {
	s, err := m.c.StartSession(options.Session())
	if err != nil {
		return nil, err
	}

	return &session{s: s}, nil
}

type session struct {
	s         mongo.Session
	txRunning atomic.Bool
}

func (s *session) BindContext(ctx context.Context) context.Context {
	return mongo.NewSessionContext(ctx, s.s)
}

func (s *session) TxnWithModel(c txn.Consistency, i txn.Isolation) txn.Txn {
	if i > txn.ReadCommitted {
		panic("unsupported isolation level")
	}
	if c > txn.CausalConsistency {
		panic("unsupported consistency model")
	}

	w, r := writeconcern.Majority(), readconcern.Available()
	if i == txn.ReadCommitted {
		r = readconcern.Majority()
	}

	return &mongoTxn{
		readCon:  r,
		writeCon: w,
	}
}

func (s *session) Txn() txn.Txn {
	return s.TxnWithModel(txn.CausalConsistency, txn.ReadCommitted)
}

func (s *session) Close(ctx context.Context) {
	s.s.EndSession(ctx)
}

type mongoTxn struct {
	readCon  *readconcern.ReadConcern
	writeCon *writeconcern.WriteConcern
	finished bool
}

func (m *mongoTxn) Start(ctx context.Context) error {
	return mongo.SessionFromContext(ctx).
		StartTransaction(
			options.Transaction().
				SetReadConcern(m.readCon).
				SetWriteConcern(m.writeCon),
		)
}

func (m *mongoTxn) Abort(ctx context.Context) error {
	err := mongo.SessionFromContext(ctx).AbortTransaction(ctx)
	m.finished = true
	return err
}

func (m *mongoTxn) Commit(ctx context.Context) error {
	err := mongo.SessionFromContext(ctx).CommitTransaction(ctx)
	m.finished = true
	return err
}

func (m *mongoTxn) Close(ctx context.Context) error {
	if !m.finished {
		return errors.WrapFail(m.Abort(ctx), "abort running txn")
	}
	return nil
}
