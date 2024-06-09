package txn

import "context"

type Session interface {
	BindContext(ctx context.Context) context.Context
	Close(ctx context.Context)
	Txn() Txn
}

type Txn interface {
	SetModel(model ConsistencyModel) Txn
	SetIsolation(lvl IsolationLevel) Txn

	Start(ctx context.Context) (ActiveTxn, error)
}

type ActiveTxn interface {
	Abort(ctx context.Context) error
	Commit(ctx context.Context) error
	Close(ctx context.Context) error
}

type ConsistencyModel int

const (
	// CausalConsistency means that
	// all logically depending operations
	// are sequential consistent
	CausalConsistency ConsistencyModel = iota

	// SequentialConsistency means that
	// any concurrent operations execution
	// result is equivalent to some
	// sequential execution of those
	// operations
	SequentialConsistency

	// Linearizable means that
	// operations order is consistent
	// with real time order
	Linearizable
)

type IsolationLevel int

const (
	ReadUncommitted IsolationLevel = iota
	ReadCommitted
	SnapshotIsolation
	Serializable
)
