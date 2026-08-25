package proxmox

import (
	"context"
	"errors"
)

type OpenIDAuthURLResponse string

type OpenIDLoginResponse struct {
	Ticket              string `json:"ticket,omitempty"`
	CSRFPreventionToken string `json:"CSRFPreventionToken,omitempty"`
	Username            string `json:"username,omitempty"`
	Cap                 any    `json:"cap,omitempty"`
	ClusterName         string `json:"clustername,omitempty"`
}

func (c *Client) OpenIDAuthURL(ctx context.Context, realm, redirectURL string) (string, error) {
	if realm == "" || redirectURL == "" {
		return "", errors.New("realm and redirect-url are required")
	}
	body := map[string]string{
		"realm":        realm,
		"redirect-url": redirectURL,
	}
	var url string
	if err := c.Post(ctx, "/access/openid/auth-url", body, &url); err != nil {
		return "", err
	}
	return url, nil
}

func (c *Client) OpenIDLogin(ctx context.Context, code, state, redirectURL string) (*OpenIDLoginResponse, error) {
	if code == "" || state == "" || redirectURL == "" {
		return nil, errors.New("code, state, and redirect-url are required")
	}
	body := map[string]string{
		"code":         code,
		"state":        state,
		"redirect-url": redirectURL,
	}
	resp := &OpenIDLoginResponse{}
	if err := c.Post(ctx, "/access/openid/login", body, resp); err != nil {
		return nil, err
	}
	return resp, nil
}
