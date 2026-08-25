package proxmox

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type Option func(*Client)

func WithClient(client *http.Client) Option {
	return WithHTTPClient(client)
}

func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) {
		c.httpClient = client
	}
}

func WithLogins(username, password string) Option {
	return WithCredentials(&Credentials{
		Username: username,
		Password: password,
	})
}

func WithCredentials(credentials *Credentials) Option {
	return func(c *Client) {
		c.credentials = credentials
	}
}

func WithAPIToken(tokenID, secret string) Option {
	return func(c *Client) {
		c.token = fmt.Sprintf("%s=%s", tokenID, secret)
	}
}

func WithSession(ticket, csrfPreventionToken string) Option {
	return func(c *Client) {
		c.session = &Session{
			Ticket:              ticket,
			CSRFPreventionToken: csrfPreventionToken,
		}
	}
}

func WithUserAgent(ua string) Option {
	return func(c *Client) {
		c.userAgent = ua
	}
}

func WithLogger(logger LeveledLoggerInterface) Option {
	return func(c *Client) {
		c.log = logger
	}
}

func WithTimeout(d time.Duration) Option {
	return func(c *Client) {
		c.timeout = d
	}
}

func WithInsecureSkipVerify() Option {
	return func(c *Client) {
		c.tlsMods = append(c.tlsMods, func(tc *tls.Config) {
			tc.InsecureSkipVerify = true
		})
	}
}

func WithRootCAs(pool *x509.CertPool) Option {
	return func(c *Client) {
		c.tlsMods = append(c.tlsMods, func(tc *tls.Config) {
			tc.RootCAs = pool
		})
	}
}

func WithRootCAFile(path string) Option {
	return func(c *Client) {
		data, err := os.ReadFile(path)
		if err != nil {
			c.log.Errorf("WithRootCAFile: read %q: %v", path, err)
			return
		}
		c.tlsMods = append(c.tlsMods, func(tc *tls.Config) {
			if tc.RootCAs == nil {
				tc.RootCAs = x509.NewCertPool()
			}
			if !tc.RootCAs.AppendCertsFromPEM(data) {
				c.log.Errorf("WithRootCAFile: %q contained no valid PEM certificates", path)
			}
		})
	}
}

func WithClientCertificate(cert tls.Certificate) Option {
	return func(c *Client) {
		c.tlsMods = append(c.tlsMods, func(tc *tls.Config) {
			tc.Certificates = append(tc.Certificates, cert)
		})
	}
}

func WithOTP(otp string) Option {
	return func(c *Client) {
		c.otp = otp
	}
}

func WithDefaultRealm(realm string) Option {
	return func(c *Client) {
		c.defaultRealm = strings.TrimPrefix(realm, "@")
	}
}

func WithProxy(u *url.URL) Option {
	return func(c *Client) {
		c.proxyFunc = func(*http.Request) (*url.URL, error) {
			return u, nil
		}
	}
}

func WithProxyFromEnvironment() Option {
	return func(c *Client) {
		c.proxyFunc = http.ProxyFromEnvironment
	}
}

func WithEagerAuth() Option {
	return func(c *Client) {
		c.eagerAuth = true
	}
}

func (c *Client) ensureOwnHTTPClient() {
	if c.httpClient == nil || c.httpClient == http.DefaultClient {
		c.httpClient = &http.Client{}
	}
}

func (c *Client) ensureTransport() *http.Transport {
	c.ensureOwnHTTPClient()
	if c.httpClient.Transport == nil {
		c.httpClient.Transport = http.DefaultTransport.(*http.Transport).Clone()
	}
	if t, ok := c.httpClient.Transport.(*http.Transport); ok {
		return t
	}
	return nil
}

func (c *Client) ensureTLSConfig() *tls.Config {
	t := c.ensureTransport()
	if t == nil {
		return nil
	}
	if t.TLSClientConfig == nil {
		t.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	}
	return t.TLSClientConfig
}

func WithRequestInterceptor(fn func(*http.Request) error) Option {
	return func(c *Client) {
		if fn == nil {
			return
		}
		c.interceptors = append(c.interceptors, fn)
	}
}
