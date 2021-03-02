package floodnet

import (
	"testing"

	"github.com/inet256/inet256/pkg/inet256test"
)

func TestNetwork(t *testing.T) {
	inet256test.TestNetwork(t, Factory)
}

func TestServer(t *testing.T) {
	inet256test.TestServer(t, Factory)
}
