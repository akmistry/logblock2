package util

func SetDefaultIfZero[V comparable](v *V, defaultVal V) {
	var zeroVal V
	if *v == zeroVal {
		*v = defaultVal
	}
}
