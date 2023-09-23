package storage

type Indexed interface {
	ID() string
}

type Model[T Indexed] interface {
	GetData() map[string]T
	SetData(map[string]T)
}
