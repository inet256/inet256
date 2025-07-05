// Package inet256tests contains a test suite for an INET256 service.
package inet256tests

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.brendoncarroll.net/stdctx/logctx"
	"go.uber.org/zap"
)

func Context(t testing.TB) context.Context {
	l, err := zap.NewDevelopment()
	require.NoError(t, err)
	ctx := context.Background()
	ctx = logctx.NewContext(ctx, l)
	return ctx
}
