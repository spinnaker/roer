package spinnaker

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
	"github.com/robzienert/tiller/cliutil"
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

	c := http.Client{
		Timeout: 10 * time.Second,
	}

	if cliutil.ContainsAllFlags(cc, []string{"certPath", "keyPath"}) {
		cert, err := tls.LoadX509KeyPair(cc.String("certPath"), cc.String("keyPath"))
		if err != nil {
			return nil, errors.Wrap(err, "loading x509 keypair")
		}

		clientCA, err := ioutil.ReadFile(cc.String("certPath"))
		if err != nil {
			return nil, errors.Wrap(err, "loading client CA")
		}

		clientCertPool := x509.NewCertPool()
		clientCertPool.AppendCertsFromPEM(clientCA)

		tlsConfig := &tls.Config{
			Certificates:       []tls.Certificate{cert},
			RootCAs:            clientCertPool,
			InsecureSkipVerify: true,
		}

		tlsConfig.BuildNameToCertificate()

		c.Transport = &http.Transport{
			TLSClientConfig: tlsConfig,
		}
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
