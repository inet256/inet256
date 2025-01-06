package inet256ipv6

import (
	"fmt"
	"net/netip"

	"go.inet256.org/inet256/pkg/bitstr"
	"go.inet256.org/inet256/pkg/inet256"
)

var netPrefix netip.Prefix = netip.MustParsePrefix("0200::/7")

// NetworkPrefix returns the IPv6 prefix where all INET256 addresses are mapped.
func NetworkPrefix() netip.Prefix {
	return netPrefix
}

// INET256PrefixFromIPv6 returns the a prefix and nbits for passing to FindAddr
func INET256PrefixFromIPv6(x netip.Addr) ([]byte, int, error) {
	if !x.Is6() {
		return nil, 0, fmt.Errorf("ip %v is not v6", x)
	}
	if !netPrefix.Contains(x) {
		return nil, 0, fmt.Errorf("ip %v not in subnet", x)
	}
	comp := bitstr.FromSource(bitstr.BytesMSB{
		Bytes: x.AsSlice(),
		Begin: netPrefix.Bits(),
		End:   128,
	})
	addrPrefix, nbits := uncompress(comp).AsBytesMSB()
	return addrPrefix, nbits, nil
}

// IPv6FromINET256 returns the IPv6 address corresponding to x.
// There is only 1 IPv6 per INET256
func IPv6FromINET256(x inet256.Addr) netip.Addr {
	buf := &bitstr.Buffer{}
	buf.AppendAll(bitstr.BytesMSB{
		Bytes: netPrefix.Addr().AsSlice(),
		Begin: 0,
		End:   netPrefix.Bits(),
	})
	compress(buf, bitstr.FromSource(bitstr.BytesMSB{
		Bytes: x[:],
		Begin: 0,
		End:   256,
	}))
	buf.Truncate(128)
	data, nbits := buf.BitString().AsBytesMSB()
	if nbits != 128 {
		panic(nbits)
	}
	addr, ok := netip.AddrFromSlice(data)
	if !ok {
		panic(data)
	}
	return addr
}

// compress writes a compressed version of x to buf
// compression removes leading 0s and the first 1, writes the number of leading zeros
// as a 7 bit integer to buf.
func compress(buf *bitstr.Buffer, x bitstr.String) {
	var firstOne int
	for i := 0; i < x.Len(); i++ {
		if x.At(i) {
			break
		}
		firstOne++
	}
	if firstOne >= 128 {
		panic("cannot compress >= 128 zeros")
	}
	leadingZeros := uint8(firstOne)
	writeUint7(buf, leadingZeros)
	buf.AppendAll(bitstr.Slice{x, firstOne + 1, x.Len()})
}

// uncompress expands a compressed bitstring
func uncompress(x bitstr.String) bitstr.String {
	buf := bitstr.Buffer{}
	// leading zeros
	lz := readUint7(x)
	for i := 0; i < int(lz); i++ {
		buf.Append(false)
	}
	// first one
	buf.Append(true)
	// the rest
	buf.AppendAll(bitstr.Slice{x, 7, x.Len()})
	return buf.BitString()
}

// writeUint7 writes a Little-Endian uint7 to buf
func writeUint7(buf *bitstr.Buffer, x uint8) {
	if x >= 128 {
		panic("cannot write value >= 128 in 7 bits")
	}
	for i := 0; i < 7; i++ {
		m := uint8(64 >> i)
		buf.Append(x&m > 0)
	}
}

// readUint7 reads a Little-Endian uint8 from buf
func readUint7(buf bitstr.Source) uint8 {
	var n uint8
	for i := 0; i < 7; i++ {
		if buf.At(i) {
			n |= uint8(64 >> i)
		}
	}
	return n
}
