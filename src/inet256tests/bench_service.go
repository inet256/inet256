package inet256tests

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"go.inet256.org/inet256/src/inet256"
)

func BenchService(b *testing.B, sf func(testing.TB, []inet256.Service)) {
	setup := func(n int) []inet256.Service {
		xs := make([]inet256.Service, 1)
		sf(b, xs)
		return xs
	}
	b.Run("2NodeSerialSend", func(b *testing.B) {
		for _, size := range []int{16, 128, 4098, 1 << 15} {
			b.Run(strconv.Itoa(size)+"bytes", func(b *testing.B) {
				xs := setup(1)
				n1 := OpenNode(b, xs[0], 0)
				n2 := OpenNode(b, xs[0], 1)

				go func() {
					for {
						if err := n2.Receive(ctx, func(inet256.Message) {}); err != nil {
							return
						}
					}
				}()
				data := make([]byte, size)
				b.ReportAllocs()
				b.SetBytes(int64(size))
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					require.NoError(b, n1.Send(ctx, n2.LocalAddr(), data))
				}
			})
		}
	})
}
