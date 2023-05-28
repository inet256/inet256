package slices2

func Map[A, B any, SA ~[]A](as SA, fn func(A) B) []B {
	bs := make([]B, len(as))
	for i := range as {
		bs[i] = fn(as[i])
	}
	return bs
}
