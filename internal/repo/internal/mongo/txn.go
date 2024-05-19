package repo

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"

	"github.com/nikmy/meowbot/pkg/errors"
	"github.com/nikmy/meowbot/pkg/txn"
)

func (m *mongoClient) NewSession() (txn.Session, error) {
	s, err := m.c.StartSession()
	if err != nil {
		return nil, err
	}

	return &session{s: s}, nil
}

type session struct {
	s mongo.Session
}

func (s *session) TxnWithModel(model txn.Model) txn.Txn {
	if model > txn.ModelSnapshotIsolation {
		panic("unsupported consistency level")
	}
	return s.Txn()
}

func (s *session) Txn() txn.Txn {
	return &mongoTxn{s: s.s}
}

func (s *session) Close(ctx context.Context) {
	s.s.EndSession(ctx)
}

type mongoTxn struct {
	s        mongo.Session
	finished bool
}

func (m *mongoTxn) Start(ctx context.Context) error {
	return errors.WrapFail(m.s.StartTransaction(), "start txn")
}

func (m *mongoTxn) Abort(ctx context.Context) error {
	err := m.s.CommitTransaction(ctx)
	if err != nil {
		return errors.WrapFail(err, "abort txn")
	}

	m.finished = true
	return nil
}

func (m *mongoTxn) Commit(ctx context.Context) error {
	err := m.s.CommitTransaction(ctx)
	if err != nil {
		return errors.WrapFail(err, "commit txn")
	}

	m.finished = true
	return nil
}

func (m *mongoTxn) Close(ctx context.Context) error {
	if !m.finished {
		return m.s.AbortTransaction(ctx)
	}
	return nil
}
