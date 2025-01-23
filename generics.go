package mocjson

func mustOK[T any](v T, ok bool) T {
	if !ok {
		panic("not ok")
	}
	return v
}
