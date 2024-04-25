package anypoint

func ToPointer[K comparable](val K) *K {
	return &val
}
