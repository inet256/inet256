package inet256http

type OpenReq struct {
	PrivateKey []byte `json:"private_key"`
}

type DropReq struct {
	PrivateKey []byte `json:"private_key"`
}
