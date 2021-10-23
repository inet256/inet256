package inet256http

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/brendoncarroll/go-p2p"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/serde"
)

type MTUResp struct {
	MTU int `json:"mtu"`
}

type FindAddrReq struct {
	Prefix []byte `json:"prefix"`
	Nbits  int    `json:"nbits"`
}

type FindAddrRes struct {
	Addr inet256.Addr `json:"addr"`
}

type NodeInfo struct {
	PublicKeyX509 []byte         `json:"public_key_x509"`
	OneHop        []inet256.Addr `json:"one_hop"`
}

const PrivateKeyHeader = "X-Private-Key"

func getPrivateKey(r *http.Request) (p2p.PrivateKey, error) {
	data, err := base64.URLEncoding.DecodeString(r.Header.Get(PrivateKeyHeader))
	if err != nil {
		return nil, err
	}
	return serde.ParsePrivateKey(data)
}

func setPrivateKey(r *http.Request, privateKey p2p.PrivateKey) {
	b64Str := base64.URLEncoding.EncodeToString(serde.MarshalPrivateKey(privateKey))
	r.Header.Add(PrivateKeyHeader, b64Str)
}

func readJSON(r io.ReadCloser, x interface{}) error {
	defer r.Close()
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &x)
}

func marshalJSON(x interface{}) []byte {
	data, err := json.Marshal(x)
	if err != nil {
		panic(err)
	}
	return data
}
