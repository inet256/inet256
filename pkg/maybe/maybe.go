package maybe

type Maybe[T any] struct {
	Ok bool
	X  T
}

func Just[T any](x T) Maybe[T] {
	return Maybe[T]{Ok: true, X: x}
}

func Nothing[T any]() Maybe[T] {
	return Maybe[T]{}
}
