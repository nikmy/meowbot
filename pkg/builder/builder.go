package builder

type Obj struct {
	A1, A2 int
}

func New[T any]() *Builder[T] {
	return &Builder[T]{
		Obj: new(T),
	}
}

type Builder[T any] struct {
	Obj *T
	Err error
}

func (b *Builder[T]) Use(setter func(b *T)) *Builder[T] {
	setter(b.Obj)
	return b
}

func (b *Builder[T]) MaybeUse(setter func(b *T) error) *Builder[T] {
	if b.Err == nil {
		b.Err = setter(b.Obj)
	}
	return b
}

func (b *Builder[T]) Get() (*T, error) {
	return b.Obj, b.Err
}
