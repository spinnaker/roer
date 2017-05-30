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
	// ErrInvalidPipelineTemplate is returned when a plan or run fails due to an
	// invalid template or configuration.
	ErrInvalidPipelineTemplate = errors.New("pipeline template is invalid")
)

// ClientConfig is used to initialize the Client
type ClientConfig struct {
	HTTPClientFactory HTTPClientFactory
	Endpoint          string
}

// Client is the Spinnaker API client
// TODO rz - this interface is pretty bad
type Client interface {
	PublishTemplate(template map[string]interface{}, update bool) (*TaskRefResponse, error)
	Plan(configuration map[string]interface{}, template map[string]interface{}) ([]byte, error)
	// Run(configuration interface{}) ([]byte, error)
	GetTask(refURL string) (*ExecutionResponse, error)
	PollTaskStatus(refURL string, timeout time.Duration) (*ExecutionResponse, error)
	GetPipelineConfig(app, pipelineConfigID string) (*PipelineConfig, error)
	SavePipelineConfig(pipelineConfig PipelineConfig) error
}

type client struct {
	endpoint   string
	httpClient *http.Client
}

// New creates a new Spinnaker client
func New(endpoint string, hc *http.Client) Client {
	return &client{endpoint: endpoint, httpClient: hc}
}

func (c *client) startPipelineURL() string {
	return c.endpoint + "/pipelines/start"
}

func (c *client) pipelineTemplatesURL() string {
	return c.endpoint + "/pipelineTemplates"
}

func (c *client) pipelineConfigsURL(app string) string {
	return c.endpoint + fmt.Sprintf("/applications/%s/pipelineConfigs", app)
}

func (c *client) pipelineConfigURL(app, pipelineConfigID string) string {
	return c.pipelineConfigsURL(app) + "/" + pipelineConfigID
}

func (c *client) pipelinesURL() string {
	return c.endpoint + "/pipelines"
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

func (c *client) Plan(configuration map[string]interface{}, template map[string]interface{}) ([]byte, error) {
	body := templatedPipelineRequest{
		Type:     "templatedPipeline",
		Config:   configuration,
		Template: template,
		Plan:     true,
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

func (c *client) GetPipelineConfig(app, pipelineConfigID string) (*PipelineConfig, error) {
	url := c.pipelineConfigURL(app, pipelineConfigID)
	logrus.WithField("url", url).Debug("getting url")
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, errors.Wrap(err, "getting pipeline config")
	}

	defer func() {
		if cerr := resp.Body.Close(); cerr != nil && err != nil {
			err = errors.Wrapf(err, "failed to close response body from %s", url)
		}
	}()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read response body from url %s", url)
	}

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return nil, nil
		}
		fmt.Println(string(respBody))
		return nil, errors.New("get pipeline config failed")
	}

	// TODO rz - HACK: Spinnaker bug returning 200 on a pipeline config that isn't found
	if len(respBody) == 0 {
		return nil, nil
	}

	var config PipelineConfig
	if err := json.Unmarshal(respBody, &config); err != nil {
		fmt.Println(string(respBody))
		return nil, errors.Wrap(err, "failed unmarshaling pipeline config response")
	}

	return &config, nil
}

func (c *client) SavePipelineConfig(pipelineConfig PipelineConfig) error {
	url := c.pipelinesURL()
	logrus.WithField("url", url).Debug("saving pipeline")
	resp, respBody, err := c.postJSON(url, pipelineConfig)
	if err != nil {
		return errors.Wrap(err, "save pipeline config")
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Println(respBody)
		return errors.New("plan request failed")
	}

	return nil
}
