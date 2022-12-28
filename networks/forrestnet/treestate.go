package forrestnet

import (
	"bytes"
	"sync"

	"github.com/brendoncarroll/go-p2p/p/kademlia"
	"github.com/brendoncarroll/go-tai64"

	"github.com/inet256/inet256/pkg/inet256"
)

const (
	beaconPurpose        = "inet256/forrestnet/chain"
	locationClaimPurpose = "inet256/forrestnet/location-claim"
)

type TreeState struct {
	privateKey  inet256.PrivateKey
	localID     Addr
	optimalRoot Addr

	mu          sync.Mutex
	currentRoot Addr
	timestamp   Timestamp
	upstream    *Peer
	beacon      *Beacon
}

func NewTreeState(privateKey inet256.PrivateKey, optimalRoot Addr) *TreeState {
	localID := inet256.NewAddr(privateKey.Public())
	return &TreeState{
		privateKey:  privateKey,
		optimalRoot: optimalRoot,
		localID:     localID,

		currentRoot: localID,
	}
}

func (ts *TreeState) Deliver(from Peer, b *Beacon, downstream []Peer) (beacons []*Beacon, _ error) {
	pubKey, err := inet256.ParsePublicKey(b.RootKey)
	if err != nil {
		return nil, err
	}
	root := inet256.NewAddr(pubKey)
	if err := VerifyBeacon(b, from.Addr); err != nil {
		return nil, err
	}
	ts.mu.Lock()
	defer ts.mu.Unlock()
	currentDist := distance(ts.optimalRoot, ts.currentRoot)
	newDist := distance(ts.optimalRoot, root)
	if bytes.Compare(newDist[:], currentDist[:]) > 0 {
		if ts.beacon == nil {
			return nil, nil
		}
		b2, err := CreateBeacon(ts.beacon, ts.privateKey, from.Index, from.PublicKey)
		if err != nil {
			return nil, err
		}
		return []*Beacon{b2}, nil
	}
	timestamp, err := tai64.ParseN(b.Timestamp)
	if err != nil {
		return nil, err
	}
	if ts.currentRoot != root {
		ts.currentRoot = root
		ts.upstream = &from
	} else if !timestamp.After(ts.timestamp) {
		return nil, nil
	}
	ts.timestamp = timestamp
	ts.beacon = b
	for _, peer := range downstream {
		b2, err := CreateBeacon(b, ts.privateKey, peer.Index, peer.PublicKey)
		if err != nil {
			return nil, err
		}
		beacons = append(beacons, b2)
	}
	return beacons, nil
}

func (ts *TreeState) OptimalRoot() Addr {
	return ts.optimalRoot
}

// Root returns the current root
func (ts *TreeState) Root() Addr {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	return ts.currentRoot
}

func (ts *TreeState) Heartbeat(now Timestamp, peers []Peer) (beacons []*Beacon) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	if ts.currentRoot != ts.localID {
		return nil
	}
	tsBytes := now.Marshal()
	base := &Beacon{
		Optimum:   ts.optimalRoot[:],
		RootKey:   inet256.MarshalPublicKey(nil, ts.privateKey.Public()),
		Timestamp: tsBytes[:],
	}
	for _, peer := range peers {
		b, err := CreateBeacon(base, ts.privateKey, peer.Index, peer.PublicKey)
		if err != nil {
			panic(err)
		}
		beacons = append(beacons, b)
	}
	return beacons
}

// LocationClaim returns a location claim for the current position in the tree.
func (ts *TreeState) LocationClaim() *LocationClaim {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	return CreateLocationClaim(ts.privateKey, ts.beacon)
}

func distance(a, b inet256.Addr) (dist [32]byte) {
	kademlia.XORBytes(dist[:], a[:], b[:])
	return dist
}

type Peer struct {
	PublicKey inet256.PublicKey
	Addr      inet256.Addr
	Index     uint32
}

type TreeLocation struct {
	Root inet256.ID
	Path []uint32
}
