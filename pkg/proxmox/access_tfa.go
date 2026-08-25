package proxmox

import (
	"context"
	"errors"
	"fmt"
)

type TFAUserEntry struct {
	UserID    string         `json:"userid,omitempty"`
	Entries   []TFAEntryInfo `json:"entries,omitempty"`
	TOTP      bool           `json:"totp,omitempty"`
	YubicoOTP bool           `json:"yubico,omitempty"`
	U2F       bool           `json:"u2f,omitempty"`
	Webauthn  bool           `json:"webauthn,omitempty"`
	Recovery  []int          `json:"recovery,omitempty"`
}

type TFAEntryInfo struct {
	ID          string    `json:"id,omitempty"`
	Type        string    `json:"type,omitempty"`
	Description string    `json:"description,omitempty"`
	Created     int64     `json:"created,omitempty"`
	Enable      IntOrBool `json:"enable,omitempty"`
}

type TFAEntryOptions struct {
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
	TOTP        string `json:"totp,omitempty"`
	Value       string `json:"value,omitempty"`
	Challenge   string `json:"challenge,omitempty"`
	Password    string `json:"password,omitempty"`
}

type TFAEntryUpdateOptions struct {
	Enable      *bool  `json:"enable,omitempty"`
	Description string `json:"description,omitempty"`
	Password    string `json:"password,omitempty"`
}

func (c *Client) TFAUsers(ctx context.Context) (users []*TFAUserEntry, err error) {
	err = c.Get(ctx, "/access/tfa", &users)
	return
}

func (c *Client) TFAEntries(ctx context.Context, userid string) (entries []*TFAEntryInfo, err error) {
	if userid == "" {
		err = errors.New("userid is required")
		return
	}
	err = c.Get(ctx, fmt.Sprintf("/access/tfa/%s", userid), &entries)
	return
}

func (c *Client) TFAEntry(ctx context.Context, userid, id string) (entry *TFAEntryInfo, err error) {
	if userid == "" || id == "" {
		err = errors.New("userid and entry id are required")
		return
	}
	err = c.Get(ctx, fmt.Sprintf("/access/tfa/%s/%s", userid, id), &entry)
	return
}

func (c *Client) NewTFAEntry(ctx context.Context, userid string, opts *TFAEntryOptions) (id string, err error) {
	if userid == "" {
		return "", errors.New("userid is required")
	}
	if opts == nil || opts.Type == "" {
		return "", errors.New("tfa entry type is required")
	}
	var resp struct {
		ID string `json:"id"`
	}
	if err = c.Post(ctx, fmt.Sprintf("/access/tfa/%s", userid), opts, &resp); err != nil {
		return "", err
	}
	return resp.ID, nil
}

func (c *Client) UpdateTFAEntry(ctx context.Context, userid, id string, opts *TFAEntryUpdateOptions) error {
	if userid == "" || id == "" {
		return errors.New("userid and entry id are required")
	}
	if opts == nil {
		opts = &TFAEntryUpdateOptions{}
	}
	return c.Put(ctx, fmt.Sprintf("/access/tfa/%s/%s", userid, id), opts, nil)
}

func (c *Client) DeleteTFAEntry(ctx context.Context, userid, id, password string) error {
	if userid == "" || id == "" {
		return errors.New("userid and entry id are required")
	}
	path := fmt.Sprintf("/access/tfa/%s/%s", userid, id)
	if password != "" {

		return c.Delete(ctx, path, map[string]string{"password": password})
	}
	return c.Delete(ctx, path, nil)
}

func (c *Client) UnlockUserTFA(ctx context.Context, userid string) error {
	if userid == "" {
		return errors.New("userid is required")
	}
	return c.Put(ctx, fmt.Sprintf("/access/users/%s/unlock-tfa", userid), nil, nil)
}
