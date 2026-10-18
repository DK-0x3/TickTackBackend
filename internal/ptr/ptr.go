package ptr

// Deref возвращает значение или zero-value
func Deref[T any](p *T) T {
	if p == nil {
		var v T
		return v
	}
	return *p
}
