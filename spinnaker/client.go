package spinnaker

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
)

var (
	ErrInvalidPipelineTemplate = errors.New("pipeline template is invalid")
)

type ClientConfig struct {
	HTTPClientFactory HTTPClientFactory
	Endpoint          string
}

// Client is the Spinnaker API client
// TODO rz - this interface is pretty bad
type Client interface {
	PublishTemplate(template map[string]interface{}, update bool) (*TaskRefResponse, error)
	Plan(configuration interface{}) ([]byte, error)
	// Run(configuration interface{}) ([]byte, error)
	GetTask(refURL string) (*ExecutionResponse, error)
	PollTaskStatus(refURL string, timeout time.Duration) (*ExecutionResponse, error)
}

type client struct {
	endpoint   string
	httpClient *http.Client
}

func New(endpoint string, hc *http.Client) Client {
	return &client{endpoint: endpoint, httpClient: hc}
}

func (c *client) startPipelineURL() string {
	return c.endpoint + "/pipelines/start"
}

func (c *client) pipelineTemplatesURL() string {
	return c.endpoint + "/pipelineTemplates"
}

func (c *client) PublishTemplate(template map[string]interface{}, update bool) (*TaskRefResponse, error) {
	url := c.pipelineTemplatesURL()
	if update {
		url = url + "/" + template["id"].(string)
	}

	resp, respBody, err := c.postJSON(url, template)
	if err != nil {
		return nil, errors.Wrap(err, "pipeline template publish")
	}

	if resp.StatusCode != http.StatusAccepted {
		fmt.Println(resp.StatusCode)
		fmt.Println(string(respBody))
		return nil, errors.New("create template request failed")
	}

	var ref TaskRefResponse
	if err := json.Unmarshal(respBody, &ref); err != nil {
		fmt.Println(string(respBody))
		return nil, errors.New("unmarshaling create template response")
	}

	return &ref, nil
}

func (c *client) Plan(configuration interface{}) ([]byte, error) {
	body := templatedPipelineRequest{
		Type:   "templatedPipeline",
		Config: configuration,
		Plan:   true,
	}

	resp, respBody, err := c.postJSON(c.startPipelineURL(), body)
	if err != nil {
		return nil, errors.Wrap(err, "pipeline template plan")
	}

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusBadRequest {
			return respBody, ErrInvalidPipelineTemplate
		}
		return respBody, errors.New("plan request failed")
	}

	return respBody, nil
}

func (c *client) GetTask(refURL string) (*ExecutionResponse, error) {
	resp, err := c.httpClient.Get(c.endpoint + refURL)
	if err != nil {
		return nil, errors.Wrap(err, "getting task status")
	}

	defer func() {
		if cerr := resp.Body.Close(); cerr != nil && err != nil {
			err = errors.Wrapf(err, "failed to close response body from %s", refURL)
		}
	}()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read response body from url %s", refURL)
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Println(string(respBody))
		return nil, errors.New("get task status failed")
	}

	var task ExecutionResponse
	if err := json.Unmarshal(respBody, &task); err != nil {
		fmt.Println(string(respBody))
		return nil, errors.Wrap(err, "failed unmarshaling task status response")
	}

	return &task, nil
}

func (c *client) PollTaskStatus(refURL string, timeout time.Duration) (*ExecutionResponse, error) {
	logrus.Info("Waiting for task to complete...")

	timer := time.NewTimer(timeout)
	t := time.NewTicker(1 * time.Second)
	defer t.Stop()

	for range t.C {
		resp, err := c.GetTask(refURL)
		if err != nil {
			return nil, errors.Wrap(err, "failed polling task status")
		}
		if resp.EndTime > 0 {
			return resp, nil
		}

		select {
		case <-timer.C:
			return nil, errors.New("timed out waiting for task to complete")
		default:
			logrus.WithField("status", resp.Status).Debug("Polling task")
		}
	}

	return nil, errors.New("exited poll loop before completion")
}
