package pointer

// To returns a pointer to the given value.
func To[T any](v T) *T {
	return &v
}

// Deref dereferences ptr and returns the value it points to if not nil, or else
// returns a default value (def).
func Deref[T any](ptr *T, def T) T {
	if ptr != nil {
		return *ptr
	}
	return def
}
