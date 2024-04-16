package repo



type filter struct {
	id *string
	fn *func(any) bool
}

type Filter func(*filter)

func ByID(id string) Filter {
	return func(f *filter) {
		f.id = &id
	}
}

func Where[T any](filterFunc func(T) bool) Filter {
	check := func(x any) bool {
		t, ok := x.(T)
		return ok && filterFunc(t)
	}
	return func(f *filter) {
		f.fn = &check
	}
}
