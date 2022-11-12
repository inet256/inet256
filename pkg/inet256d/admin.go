package inet256d

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/mesh256"
)

type StatusRes struct {
	MainAddr       inet256.Addr         `json:"main_addr"`
	TransportAddrs []string             `json:"transport_addrs"`
	Peers          []mesh256.PeerStatus `json:"peers"`
}

type AdminClient interface {
	GetStatus(ctx context.Context) (*StatusRes, error)
}

type adminClient struct {
	endpoint *url.URL
	hc       *http.Client
}

func NewAdminClient(endpoint string) (AdminClient, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	return &adminClient{
		endpoint: u,
		hc:       http.DefaultClient,
	}, nil
}

func (c *adminClient) GetStatus(ctx context.Context) (*StatusRes, error) {
	return doRequest[struct{}, StatusRes](ctx, c.hc, http.MethodGet, c.getPath("status"), nil)
}

func (c *adminClient) getPath(p string) string {
	u := *c.endpoint
	u.Path = strings.TrimRight(u.Path, "/") + "/" + strings.Trim(p, "/")
	return u.String()
}

func doRequest[Req, Res any](ctx context.Context, hc *http.Client, method, url string, x *Req) (*Res, error) {
	var reqBody io.Reader
	if x != nil {
		data, err := json.Marshal(x)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, err
	}
	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("non-200 status %v", resp.Status)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var res Res
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, err
	}
	return &res, nil
}
