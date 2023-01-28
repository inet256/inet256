package inet256lb

import (
	"context"
	"io"
)

type StreamMuxEndpoint interface {
	Open(ctx context.Context) (StreamMuxSession, error)
}

type StreamMuxSession interface {
	OpenStream(ctx context.Context) (io.ReadWriteCloser, error)
	AcceptStream(ctx context.Context) (io.ReadWriteCloser, error)
	Close() error
}
