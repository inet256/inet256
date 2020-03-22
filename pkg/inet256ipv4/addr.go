package inet256ipv4

import (
	"encoding/json"
	"errors"
	"net"

	"golang.org/x/net/ipv4"
)

type IPv4 [4]byte

func IPv4FromString(x string) IPv4 {
	ip := IPv4{}
	netIP := net.ParseIP(x)
	if netIP.To4() == nil {
		panic(x)
	}
	copy(ip[:], netIP.To4())
	return ip
}

func (ip *IPv4) MarshalJSON() ([]byte, error) {
	netip := (net.IP)(ip[:])
	return json.Marshal(netip.String())
}

func (ip *IPv4) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	netip := net.ParseIP(s)
	ip4 := netip.To4()
	if ip4 == nil {
		return errors.New("not ipv4 address")
	}
	copy(ip[:], ip4)
	return nil
}

const (
	offsetSrc = 12
	offsetDst = offsetSrc + 4
)

func getDst(buf []byte) IPv4 {
	ip := IPv4{}
	copy(ip[:], buf[offsetDst:])
	return ip
}

func getSrc(buf []byte) IPv4 {
	ip := IPv4{}
	copy(ip[:], buf[offsetSrc:])
	return ip
}

func getPayload(buf []byte) ([]byte, error) {
	header, err := ipv4.ParseHeader(buf)
	if err != nil {
		return nil, err
	}
	return buf[header.TotalLen:], nil
}
