package forrestnet

import (
	"bytes"
	"encoding/binary"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-tai64"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
)

type Timestamp = tai64.TAI64N

func VerifyBeacon(b *Beacon, leaf inet256.Addr) error {
	root := inet256.NewAddrPKIX(b.RootKey)
	parentPubKey, err := p2p.ParsePublicKey(b.RootKey)
	if err != nil {
		return err
	}
	if len(b.Chain) < 1 {
		return errors.Errorf("no chain in beacon")
	}
	leafPubKey, err := inet256.ParsePublicKey(b.Chain[len(b.Chain)-1].PublicKey)
	if err != nil {
		return err
	}
	actualLeaf := inet256.NewAddr(leafPubKey)
	if actualLeaf != leaf {
		return errors.Errorf("beacon not for this node HAVE: %v WANT: %v", actualLeaf, leaf)
	}
	timestamp, err := tai64.ParseN(b.Timestamp)
	if err != nil {
		return err
	}
	for _, link := range b.Chain {
		if err := VerifyLink(parentPubKey, root, timestamp, link); err != nil {
			return err
		}
		pubKey, err := p2p.ParsePublicKey(link.PublicKey)
		if err != nil {
			return err
		}
		parentPubKey = pubKey
	}
	return nil
}

func VerifyLink(parentKey p2p.PublicKey, root inet256.Addr, timestamp Timestamp, link *Link) error {
	pubKey, err := p2p.ParsePublicKey(link.PublicKey)
	if err != nil {
		return err
	}
	id := inet256.NewAddr(pubKey)
	data := prepareSigningData(root, timestamp, link.Branch, id)
	return p2p.Verify(parentKey, beaconPurpose, data, link.Sig)
}

func CreateBeacon(base *Beacon, privateKey p2p.PrivateKey, branch uint32, childKey p2p.PublicKey) (*Beacon, error) {
	root := inet256.NewAddrPKIX(base.RootKey)
	timestamp, err := tai64.ParseN(base.Timestamp)
	if err != nil {
		return nil, err
	}
	chain := append([]*Link{}, base.Chain...)
	link, err := CreateLink(privateKey, root, timestamp, branch, childKey)
	if err != nil {
		return nil, err
	}
	b := &Beacon{
		RootKey:   base.RootKey,
		Timestamp: base.Timestamp,
		Chain:     append(chain, link),
	}
	return b, nil
}

func CreateLink(privateKey inet256.PrivateKey, root inet256.Addr, timestamp Timestamp, branch uint32, childKey p2p.PublicKey) (*Link, error) {
	childID := inet256.NewAddr(childKey)
	data := prepareSigningData(root, timestamp, branch, childID)
	sig, err := p2p.Sign(nil, privateKey, beaconPurpose, data)
	if err != nil {
		return nil, err
	}
	link := &Link{
		PublicKey: p2p.MarshalPublicKey(childKey),
		Branch:    branch,
		Sig:       sig,
	}
	return link, nil
}

// CreateLocationClaim
func CreateLocationClaim(privateKey inet256.PrivateKey, b *Beacon) *LocationClaim {
	id := inet256.NewAddr(privateKey.Public())
	if bytes.Equal(b.RootKey[:], id[:]) {
		return nil
	}
	data, _ := proto.Marshal(b)
	sig, err := p2p.Sign(nil, privateKey, locationClaimPurpose, data)
	if err != nil {
		panic(err)
	}
	return &LocationClaim{
		Beacon: b,
		Sig:    sig,
	}
}

// VerifyLocationClaim
func VerifyLocationClaim(pubKey inet256.PublicKey, lc *LocationClaim) error {
	data, err := proto.Marshal(lc)
	if err != nil {
		panic(err)
	}
	return p2p.Verify(pubKey, locationClaimPurpose, data, lc.Sig)
}

func parseBeacon(x []byte) (*Beacon, error) {
	b := &Beacon{}
	if err := proto.Unmarshal(x, b); err != nil {
		return nil, err
	}
	return b, nil
}

func prepareSigningData(rootAddr inet256.Addr, timestamp Timestamp, branch uint32, id inet256.Addr) []byte {
	var data []byte
	tsBytes := timestamp.Marshal()
	data = append(data, rootAddr[:]...)
	data = append(data, tsBytes[:]...)
	data = appendUint32(data, branch)
	data = append(data, id[:]...)
	return data
}

func appendUint32(x []byte, n uint32) []byte {
	buf := [4]byte{}
	binary.BigEndian.PutUint32(buf[:], n)
	return append(x, buf[:]...)
}
