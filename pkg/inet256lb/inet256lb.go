package inet256lb

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/netip"
	"strings"

	"golang.org/x/sync/errgroup"

	"github.com/inet256/inet256/pkg/inet256"
)

func MakeStreamFrontend(protocol string, node inet256.Node) (StreamEndpoint, error) {
	switch protocol {
	case "utp":
		return NewUTPFrontend(node), nil
	default:
		return nil, fmt.Errorf("unrecognized frontend protocol %q", protocol)
	}
}

func MakeStreamBackend(x string, getNode func() inet256.Node) (StreamEndpoint, error) {
	scheme, rest, err := parseEndpoint(x)
	if err != nil {
		return nil, err
	}
	switch scheme {
	case "unix":
		return NewUNIXStreamBackend(rest), nil
	case "tcp+ip":
		ap, err := netip.ParseAddrPort(rest)
		if err != nil {
			return nil, err
		}
		return NewTCPIPBackend(ap), nil

	case "tcp":
		fallthrough
	case "tcp+inet256":
		return nil, errors.New(`TCP on INET256 is not yet supported, try "tcp+ip"`)

	case "utp":
		fallthrough
	case "utp+inet256":
		raddr, err := inet256.ParseAddrBase64([]byte(rest))
		if err != nil {
			return nil, err
		}
		return NewUTPBackend(getNode(), raddr), nil

	default:
		return nil, fmt.Errorf("unrecognized stream backend protocol: %q", scheme)
	}
}

func parseEndpoint(x string) (scheme, target string, _ error) {
	parts := strings.SplitN(x, "://", 2)
	if len(parts) < 2 {
		parts = strings.SplitN(x, ":", 2)
	}
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid endpoint: %q", x)
	}
	return parts[0], parts[1], nil
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
