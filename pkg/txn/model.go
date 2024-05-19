package txn

import "context"

type Session interface {
	BindContext(ctx context.Context) context.Context

	Txn() Txn
	TxnWithModel(c Consistency, i Isolation) Txn
	Close(ctx context.Context)
}

type Txn interface {
	Start(ctx context.Context) error
	ActiveTxn
}

type ActiveTxn interface {
	Abort(ctx context.Context) error
	Commit(ctx context.Context) error
	Close(ctx context.Context) error
}

type Consistency int

const (
	// CausalConsistency means that
	// all operations within transactions
	// are sequential consistent
	CausalConsistency Consistency = iota

	// Linearizable means that
	// operations order is consistent
	// with real time order
	Linearizable
)

type Isolation int

const (
	ReadUncommitted Isolation = iota
	ReadCommitted
	RepeatableRead
	Serializable
)
