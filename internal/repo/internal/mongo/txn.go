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
	s, err := m.c.StartSession(options.Session().SetCausalConsistency(true))
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

func (s *session) Txn() txn.Txn {
	return &mongoTxn{
		readCon:  readconcern.Majority(),
		writeCon: writeconcern.Majority(),
	}
}

func (s *session) Close(ctx context.Context) {
	s.s.EndSession(ctx)
}

type mongoTxn struct {
	readCon  *readconcern.ReadConcern
	writeCon *writeconcern.WriteConcern
	finished bool
	err      error
}

func (m *mongoTxn) SetModel(model txn.ConsistencyModel) txn.Txn {
	if m.err != nil {
		return m
	}

	if model > txn.CausalConsistency {
		m.err = errors.Error("unsupported consistency model")
	}

	return m
}

func (m *mongoTxn) SetIsolation(lvl txn.IsolationLevel) txn.Txn {
	if m.err != nil {
		return m
	}

	switch lvl {
	case txn.ReadUncommitted:
		m.readCon = readconcern.Available()
	case txn.ReadCommitted:
		m.readCon = readconcern.Majority()
	default:
		m.err = errors.Error("unsupported isolation level")
	}

	return m
}

func (m *mongoTxn) Start(ctx context.Context) (txn.ActiveTxn, error) {
	return m, mongo.SessionFromContext(ctx).
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
