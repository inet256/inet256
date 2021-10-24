package inet256ipv6

import (
	"bytes"

	"github.com/inet256/inet256/pkg/inet256"
	"github.com/pkg/errors"
)

// ParseWhitelist creates an AllowFunc which allows only the addresses listed in x.
// It expects newline separated base64 encoded INET256 addresses.  One per line.
func ParseWhitelist(x []byte) (AllowFunc, error) {
	lines := bytes.Split(x, []byte("\n"))
	allowed := make(map[inet256.Addr]struct{}, len(lines))
	for i, line := range lines {
		addr, err := inet256.ParseAddrB64(line)
		if err != nil {
			return nil, errors.Errorf("line %d: %q is not a valid INET256 address: %v", i, line, err)
		}
		allowed[addr] = struct{}{}
	}
	return func(x inet256.Addr) bool {
		_, exists := allowed[x]
		return exists
	}, nil
}
