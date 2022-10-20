package util

func Contains[T comparable](s []T, elem T) bool {
	for _, v := range s {
		if v == elem {
			return true
		}
	}

	return false
}

func Pop[T comparable](s []T) {
	s = s[:len(s)-1]
}

func Filter[T comparable](s []T, f func(T) bool) []T {
	var filtered []T
	for _, elem := range s {
		if f(elem) {
			filtered = append(filtered, elem)
		}
	}
	return filtered
}
