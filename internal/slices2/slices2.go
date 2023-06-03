package slices2

func Map[A, B any, SA ~[]A](as SA, fn func(A) B) []B {
	bs := make([]B, len(as))
	for i := range as {
		bs[i] = fn(as[i])
	}
	return bs
}

func Filter[T any, S ~[]T](xs S, fn func(T) bool) S {
	ret := xs[:0]
	for i := range xs {
		if fn(xs[i]) {
			ret = append(ret, xs[i])
		}
	}
	return ret
}
