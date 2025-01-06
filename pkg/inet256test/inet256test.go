package inet256test

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
