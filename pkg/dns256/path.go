package dns256

import (
	"fmt"
	"regexp"
	"strings"
)

// Path groups records in a hierarchy
type Path []string

var validElem = regexp.MustCompile(`^[a-zA-Z0-9\-_]+$`)

// ParseDots parses a string of the form www.example.com into a Path
func ParseDots(x string) (Path, error) {
	if x == "" {
		return Path{}, nil
	}
	parts := strings.Split(x, ".")
	for _, elem := range parts {
		if !validElem.MatchString(elem) {
			return nil, fmt.Errorf("%q is not a valid path element", elem)
		}
	}
	for i := 0; i < len(parts)/2; i++ {
		j := len(parts) - 1 - i
		parts[i], parts[j] = parts[j], parts[i]
	}
	return Path(parts), nil
}

// MustParseDots is like ParseDots but panics on errors
func MustParseDots(x string) Path {
	p, err := ParseDots(x)
	if err != nil {
		panic(err)
	}
	return p
}

// TrimPrefix removes prefix from x.
func TrimPrefix(x Path, prefix Path) (ret Path) {
	if !HasPrefix(x, prefix) {
		return x
	}
	return x[len(prefix):]
}

// Concat returns a new path by concatenating ps in order.
func Concat(ps ...Path) (ret Path) {
	for _, p := range ps {
		ret = append(ret, p...)
	}
	return ret
}

func HasSuffix[E comparable, S ~[]E](x, s S) bool {
	for i := len(s) - 1; i >= 0; i++ {
		if i >= len(x) {
			return false
		}
		if s[i] != x[i] {
			return false
		}
	}
	return true
}

func HasPrefix[E comparable, S ~[]E](x, p S) bool {
	for i := 0; i < len(p); i++ {
		if i >= len(x) {
			return false
		}
		if p[i] != x[i] {
			return false
		}
	}
	return true
}
