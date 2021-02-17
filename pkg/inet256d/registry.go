package inet256d

import (
	"encoding/binary"

	"github.com/inet256/inet256/pkg/inet256"
)

var (
	networkIndexes = map[uint64]inet256.NetworkSpec{}
	networkNames   = map[string]inet256.NetworkSpec{}
)

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

func Register(index uint64, name string, factory inet256.NetworkFactory) {
	netSpec := inet256.NetworkSpec{
		Name:    name,
		Index:   index,
		Factory: factory,
	}
	if _, exists := networkIndexes[netSpec.Index]; exists {
		panic("network by that name already exists")
	}
	if _, exists := networkNames[netSpec.Name]; exists {
		panic("network already exists at that index")
	}
	networkIndexes[netSpec.Index] = netSpec
	networkNames[netSpec.Name] = netSpec
}
