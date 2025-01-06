package inet256d

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path"

	"go.inet256.org/inet256/pkg/inet256"
	"go.inet256.org/inet256/pkg/mesh256"
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
	d        net.Dialer
}

func NewAdminClient(endpoint string) (AdminClient, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	return &adminClient{
		endpoint: u,
	}, nil
}

func (c *adminClient) GetStatus(ctx context.Context) (*StatusRes, error) {
	conn, err := c.dial(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	return doRequest[struct{}, StatusRes](ctx, conn, http.MethodGet, c.getPath("status"), nil)
}

func (c *adminClient) getPath(p string) string {
	return path.Join("/admin", p)
}

func (c *adminClient) dial(ctx context.Context) (net.Conn, error) {
	return c.d.DialContext(ctx, c.endpoint.Scheme, path.Join(c.endpoint.Host, c.endpoint.Path))
}

func doRequest[Req, Res any](ctx context.Context, c net.Conn, method, url string, x *Req) (*Res, error) {
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
	if err := req.Write(c); err != nil {
		return nil, err
	}
	bufr := bufio.NewReader(c)
	resp, err := http.ReadResponse(bufr, req)
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
