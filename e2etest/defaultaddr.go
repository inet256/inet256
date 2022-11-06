package main

import (
	"testing"

	"github.com/inet256/inet256/client/go_client/inet256client"
	"github.com/inet256/inet256/pkg/inet256d"
	"github.com/stretchr/testify/require"
)

func TestDefaultAddr(t *testing.T) {
	require.Equal(t, inet256client.DefaultAPIEndpoint, inet256d.DefaultAPIEndpoint)
}
