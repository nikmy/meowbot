package repo

func newFilter() *filter {
	return &filter{fields: map[string]any{}}
}

type filter struct {
	fields map[string]any

	id *string
	fn *func(any) bool
}

type Filter func(*filter)

func ByID(id string) Filter {
	return func(f *filter) {
		f.id = &id
	}
}

func ByField(field string, value any) Filter {
	return func(f *filter) {
		f.fields[field] = value
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
