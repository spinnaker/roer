package spinnaker

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

// HTTPClientFactory creates a new http.Client from the cli.Context
type HTTPClientFactory func(cc *cli.Context) (*http.Client, error)

// DefaultHTTPClientFactory creates a basic http.Client that by default can
// take an x509 cert/key pair for API authentication.
func DefaultHTTPClientFactory(cc *cli.Context) (*http.Client, error) {
	if cc == nil {
		logrus.Panic("cli context has not been set")
	}
	var c http.Client
	cookieJar, _ := cookiejar.New(nil)

	if cc.GlobalIsSet("apiSession") {
		var cookies []*http.Cookie
		cookie := &http.Cookie{
			Name:  "SESSION",
			Value: cc.GlobalString("apiSession"),
		}
		cookies = append(cookies, cookie)
		u, _ := url.Parse(os.Getenv("SPINNAKER_API"))
		cookieJar.SetCookies(u, cookies)
	}

	c = http.Client{
		Timeout: time.Duration(cc.GlobalInt("clientTimeout")) * time.Second,
		Jar:     cookieJar,
	}

	var certPath string
	var keyPath string

	if cc.GlobalIsSet("certPath") {
		certPath = cc.GlobalString("certPath")
	} else if os.Getenv("SPINNAKER_CLIENT_CERT") != "" {
		certPath = os.Getenv("SPINNAKER_CLIENT_CERT")
	} else {
		certPath = ""
	}
	if cc.GlobalIsSet("keyPath") {
		keyPath = cc.GlobalString("keyPath")
	} else if os.Getenv("SPINNAKER_CLIENT_KEY") != "" {
		keyPath = os.Getenv("SPINNAKER_CLIENT_KEY")
	} else {
		keyPath = ""
	}

	c.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{},
	}

	if certPath != "" && keyPath != "" {
		logrus.Debug("Configuring TLS with pem cert/key pair")
		cert, err := tls.LoadX509KeyPair(certPath, keyPath)
		if err != nil {
			return nil, errors.Wrap(err, "loading x509 keypair")
		}

		clientCA, err := ioutil.ReadFile(certPath)
		if err != nil {
			return nil, errors.Wrap(err, "loading client CA")
		}

		clientCertPool := x509.NewCertPool()
		clientCertPool.AppendCertsFromPEM(clientCA)

		c.Transport.(*http.Transport).TLSClientConfig.MinVersion = tls.VersionTLS12
		c.Transport.(*http.Transport).TLSClientConfig.PreferServerCipherSuites = true
		c.Transport.(*http.Transport).TLSClientConfig.Certificates = []tls.Certificate{cert}
		c.Transport.(*http.Transport).TLSClientConfig.InsecureSkipVerify = true
	}

	if cc.GlobalIsSet("insecure") {
		c.Transport.(*http.Transport).TLSClientConfig.InsecureSkipVerify = true
	}

	return &c, nil
}

func (c *client) postJSON(url string, body interface{}) (resp *http.Response, respBody []byte, err error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, nil, errors.Wrap(err, "marshaling body to json")
	}
	resp, err = c.httpClient.Post(url, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return nil, nil, errors.Wrapf(err, "posting to %s", url)
	}

	defer func() {
		if cerr := resp.Body.Close(); cerr != nil && err != nil {
			err = errors.Wrapf(err, "failed to close response body from %s", url)
		}
	}()

	respBody, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to read response body from url %s", url)
	}

	return resp, respBody, nil
}

func (c *client) postForm(url string, data url.Values) (resp *http.Response, respBody []byte, err error) {
	resp, err = c.httpClient.PostForm(url, data)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "posting to %s", url)
	}

	defer func() {
		if cerr := resp.Body.Close(); cerr != nil && err != nil {
			err = errors.Wrapf(err, "failed to close response body from %s", url)
		}
	}()

	respBody, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to read response body from url %s", url)
	}

	return resp, respBody, nil
}

func (c *client) getJSON(url string) (resp *http.Response, respBody []byte, err error) {
	resp, err = c.httpClient.Get(url)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "posting to %s", url)
	}

	defer func() {
		if cerr := resp.Body.Close(); cerr != nil && err != nil {
			err = errors.Wrapf(err, "failed to close response body from %s", url)
		}
	}()

	respBody, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to read response body from url %s", url)
	}

	return resp, respBody, nil
}

func (c *client) delete(url string) (resp *http.Response, respBody []byte, err error) {
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to create delete request object")
	}

	resp, err = c.httpClient.Do(req)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to make delete request to %s", url)
	}

	defer func() {
		if cerr := resp.Body.Close(); cerr != nil && err != nil {
			err = errors.Wrapf(err, "failed to close response body from %s", url)
		}
	}()

	respBody, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to read response body from url %s", url)
	}

	return resp, respBody, nil
}
