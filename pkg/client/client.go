package client

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"path"
	"time"

	"github.com/jdtw/links/pkg/auth"
	pb "github.com/jdtw/links/proto/links"
	"google.golang.org/protobuf/proto"
)

const linksAPI = "/api/links"

// Client is a client for the links REST API.
type Client struct {
	Host string
	// If PKCS8 is not nil, the client sends unauthenticated requests.
	PKCS8  []byte
	Client *http.Client
}

// New creates a client with a default HTTP client. If pkcs8 is nil,
// client requests will be unauthenticated.
func New(host string, pkcs8 []byte) *Client {
	return &Client{
		Host:   host,
		PKCS8:  pkcs8,
		Client: &http.Client{},
	}
}

func (c *Client) do(method string, path string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, c.Host+path, body)
	if err != nil {
		return nil, err
	}
	if len(c.PKCS8) > 0 {
		token, err := auth.SignJWT(c.PKCS8, auth.ClientAudience(req), auth.WithExpiry(time.Now(), 10*time.Second))
		if err != nil {
			return nil, err
		}
		req.Header.Add("Authorization", "Bearer "+string(token))
	}
	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	if err := ok(resp); err != nil {
		resp.Body.Close()
		return nil, err
	}
	return resp, nil
}

func (c *Client) List() (map[string]string, error) {
	resp, err := c.do("GET", linksAPI, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	lpb := &pb.Links{}
	if err := unmarshalBody(resp, lpb); err != nil {
		return nil, err
	}

	l := make(map[string]string)
	for k, v := range lpb.GetLinks() {
		l[k] = v.GetUri()
	}
	return l, nil
}

func (c *Client) Add(links map[string]string) error {
	lpb := &pb.Links{
		Links: make(map[string]*pb.Link),
	}
	for k, v := range links {
		lpb.Links[k] = &pb.Link{Uri: v}
	}
	body, err := marshal(lpb)
	if err != nil {
		return err
	}
	resp, err := c.do("POST", linksAPI, body)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func (c *Client) Get(link string) (string, error) {
	resp, err := c.do("GET", api(link), nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	lpb := &pb.Link{}
	if err := unmarshalBody(resp, lpb); err != nil {
		return "", err
	}
	return lpb.GetUri(), nil
}

func (c *Client) Put(link string, uri string) error {
	lpb := &pb.Link{Uri: uri}
	body, err := marshal(lpb)
	if err != nil {
		return err
	}
	resp, err := c.do("PUT", api(link), body)
	if err != nil {
		return nil
	}
	resp.Body.Close()
	return nil
}

func (c *Client) Delete(link string) error {
	resp, err := c.do("DELETE", api(link), nil)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func api(link string) string {
	return path.Join(linksAPI, link)
}

func marshal(m proto.Message) (io.Reader, error) {
	b, err := proto.Marshal(m)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(b), nil
}

func unmarshalBody(resp *http.Response, m proto.Message) error {
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return proto.Unmarshal(b, m)
}

func ok(resp *http.Response) error {
	switch resp.StatusCode {
	case http.StatusOK, http.StatusNoContent, http.StatusCreated:
		return nil
	default:
		return fmt.Errorf("%s %s failed: %s", resp.Request.Method, resp.Request.RequestURI, resp.Status)
	}
}
