package txn

type Model int

const (
	// SnapshotIsolation model formally means:
	// 1. within a transaction T, reads observe T
	//    most recent writes (if any)
	// 2. reads without a preceding write in T1
	//    must observe the state written by a T0,
	//    such that T0 is visible to T1, and no
	//    more recent transaction wrote to that
	//    object
	SnapshotIsolation Model = iota

	// Serializable model formally means:
	// execution of the operations of
	// concurrently executing transactions
	// produces the same effect as some
	// serial execution of them
	Serializable

	// StrictSerializable model guarantees
	// that operations take place atomically:
	// a transaction’s sub-operations do not
	// appear to interleave with sub-operations
	// from other transactions. It implies
	// serializability and linearizability
	StrictSerializable
)
