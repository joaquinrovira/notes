package fp

func Map[T1, T2 any, E ~[]T1](v E, fn func(v T1) T2) []T2 {
	next := make([]T2, len(v))
	for i := range v {
		next[i] = fn(v[i])
	}
	return next
}

func Filter[T any, E ~[]T](v E, fn func(v T) bool) []T {
	var next []T
	for i := range v {
		if fn(v[i]) {
			next = append(next, v[i])
		}
	}
	return next
}
