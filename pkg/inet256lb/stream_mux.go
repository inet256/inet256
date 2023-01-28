package inet256lb

import (
	"context"
	"io"
)

type StreamMuxBackend interface {
	Dial(ctx context.Context) (StreamMuxSession, error)
}

type StreamMuxSession interface {
	OpenStream(ctx context.Context) (io.ReadWriteCloser, error)
	AcceptStream(ctx context.Context) (io.ReadWriteCloser, error)
}

type StreamMuxFrontend interface {
	Accept(ctx context.Context) (StreamMuxSession, error)
}
