package kadsrnet

import (
	"bytes"
	"encoding/binary"
	"time"

	"github.com/brendoncarroll/go-p2p"
	"github.com/pkg/errors"
)

const (
	MaxIntoPast   = 60 * time.Second
	MaxIntoFuture = 10 * time.Second

	SigPurpose = "inet256/kad+sr"
)

func SignMessage(privateKey p2p.PrivateKey, now time.Time, m *Message) error {
	m.Timestamp = now.Unix()
	msgBytes := formatSigTarget(m)
	sig, err := p2p.Sign(privateKey, SigPurpose, msgBytes)
	if err != nil {
		return err
	}
	m.Sig = sig
	return nil
}

func VerifyMessage(publicKey p2p.PublicKey, now time.Time, m *Message) error {
	minTime := now.Add(-MaxIntoPast).Unix()
	maxTime := now.Add(MaxIntoFuture).Unix()
	if m.Timestamp < minTime {
		return errors.Errorf("timestamp is too far behind")
	}
	if m.Timestamp > maxTime {
		return errors.Errorf("timestamp is too far ahead")
	}
	msgBytes := formatSigTarget(m)
	return p2p.Verify(publicKey, SigPurpose, msgBytes, m.Sig)
}

func formatSigTarget(m *Message) []byte {
	buf := &bytes.Buffer{}
	buf.Write(m.Src)
	buf.Write(m.Dst)
	buf.Write(m.Body)
	binary.Write(buf, binary.BigEndian, m.Timestamp)
	return buf.Bytes()
}
