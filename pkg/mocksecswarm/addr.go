package mocksecswarm

import "github.com/brendoncarroll/go-p2p"

type Addr struct {
	ID   p2p.PeerID
	Addr p2p.Addr
}

func (a Addr) Key() string {
	data, err := a.MarshalText()
	if err != nil {
		panic(err)
	}
	return string(data)
}

func (a Addr) MarshalText() ([]byte, error) {
	var data []byte
	part1, err := a.ID.MarshalText()
	if err != nil {
		return nil, err
	}
	data = append(data, part1...)
	data = append(data, '@')
	part2, err := a.Addr.MarshalText()
	if err != nil {
		return nil, err
	}
	data = append(data, part2...)
	return data, nil
}
