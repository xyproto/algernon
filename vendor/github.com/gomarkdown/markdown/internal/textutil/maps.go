package textutil

// MapSet sets (*m)[k] to v, making the map first if *m is nil.
func MapSet[K comparable, V any](m *map[K]V, k K, v V) {
	if *m == nil {
		*m = make(map[K]V)
	}
	(*m)[k] = v
}
