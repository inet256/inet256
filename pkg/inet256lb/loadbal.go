package inet256lb

import (
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
)

func ParseEndpoint(x string) ([]string, error) {
	parts := strings.SplitN(x, ":", 2)
	if len(parts) < 2 {
		return errors.New("protocol must contain :")
	}
	scheme := parts[0]
	endpoint := parts[1]

	protoStack := strings.SplitN(scheme, "+", 2)
}

func MakeStreamBackend(x string) (StreamBackend, error) {
	switch {
	case "unix":
		return &UNIXBackend{}, nil
	case "tcp+ip":
		return &TCPBackend{}, nil
	default:
		return fmt.Errorf("unrecognized protocol: %q", x)
	}
}

func PlumbRWC(a, b io.ReadWriteCloser) error {
	eg := errgroup.Group{}
	eg.Go(func() error {
		defer a.Close()
		_, err := io.Copy(a, b)
		if errors.Is(err, net.ErrClosed) {
			err = nil
		}
		return err
	})
	eg.Go(func() error {
		defer b.Close()
		_, err := io.Copy(b, a)
		if errors.Is(err, net.ErrClosed) {
			err = nil
		}
		return err
	})
	return eg.Wait()
}
