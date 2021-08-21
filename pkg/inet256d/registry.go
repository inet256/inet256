package inet256d

import (
	"encoding/binary"

	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/inet256srv"
)

var name2Network = map[string]inet256srv.NetworkSpec{}
var index2Network = map[uint64]string{}

func Register(name string, index uint64, nf inet256.NetworkFactory) {
	_, exists := name2Network[name]
	if exists {
		panic("duplicate network " + name)
	}
	_, exists = index2Network[index]
	if exists {
		panic("duplicate index for network" + name)
	}
	name2Network[name] = inet256srv.NetworkSpec{
		Factory: nf,
		Index:   index,
		Name:    name,
	}
	index2Network[index] = name
}

func IndexFromString(x string) uint64 {
	if len(x) > 8 {
		panic("string too long")
	}
	b := []byte(x)
	for len(b) < 8 {
		b = append(b, 0x00)
	}
	return binary.LittleEndian.Uint64(b)
}
