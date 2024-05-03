package util

func SliceLast[S ~[]E, E any](s S) E {
	return s[len(s)-1]
}

func SlicePopLast[S ~[]E, E any](s S) S {
	var zeroVal E
	s[len(s)-1] = zeroVal
	return s[:len(s)-1]
}
