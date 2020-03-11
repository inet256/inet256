package inet256ipv6

import (
	"net"

	"github.com/inet256/inet256/pkg/inet256"
)

var Subnet net.IPNet

func init() {
	_, ipnet, err := net.ParseCIDR("0200::/7")
	if err != nil {
		panic(err)
	}
	Subnet = *ipnet
}

func IPv6ToPrefix(x net.IP) ([]byte, int) {
	if !Subnet.Contains(x) {
		panic("ip not in subnet")
	}
	bitOff, _ := Subnet.Mask.Size()

	c := make([]byte, 16)
	n := bitCopy(c, 0, x, bitOff)

	return uncompress(c[:n/8])
}

func INet256ToIPv6(x inet256.Addr) net.IP {
	ip6 := IPv6Addr{}
	copy(ip6[:], Subnet.IP)

	bitOff, _ := Subnet.Mask.Size()
	c := compress(x[:])
	bitCopy(ip6[:], bitOff, c, 0)

	return (net.IP)(ip6[:])
}

func compress(x []byte) []byte {
	lz := leading0s(x)
	if lz > 255 {
		panic("cannot compress address")
	}

	y := make([]byte, len(x)-(lz+1)/8)
	y[0] = uint8(lz)
	bitCopy(y, 8, x, lz+1) // lz+1 skips first 1
	return y
}

func uncompress(x []byte) ([]byte, int) {
	lz := int(x[0])
	y := make([]byte, ceilDiv(lz+1, 8)+len(x)-1)

	for i := 0; i < lz; i++ {
		y[i/8] = 0 // set to 0
	}
	y[lz/8] |= 0x80 >> (lz % 8) // set the first 1
	n := bitCopy(y, lz+1, x, 8) // copy the rest
	l := n + lz + 1
	return y, l
}

func bitCopy(dst []byte, doff int, src []byte, soff int) int {
	di := doff
	si := soff
	for di/8 < len(dst) && si/8 < len(src) {
		// construct masks
		var sm uint8 = 0x80 >> (si % 8)
		var dm uint8 = 0x80 >> (di % 8)

		if (src[si/8] & sm) > 0 {
			dst[di/8] |= dm // if the bit is set, OR the mask
		} else {
			dst[di/8] &= (^dm) // if the bit is not set, AND the inverse mask
		}
		di++
		si++
	}
	n := di - doff
	return n
}

func ceilDiv(x int, z int) int {
	if x%z > 0 {
		return (x / z) + 1
	}
	return x / z
}
