package spinnaker

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"strconv"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
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
	PublishTemplate(template map[string]interface{}, options PublishTemplateOptions) (*TaskRefResponse, error)
	ApplicationSubmitTask(app string, task Task) (*TaskRefResponse, error)
	ApplicationGet(app string) (bool, []byte, error)
	ApplicationList() ([]ApplicationInfo, error)
	Plan(configuration map[string]interface{}, template map[string]interface{}) ([]byte, error)
	DeleteTemplate(templateID string) (*TaskRefResponse, error)
	// Run(configuration interface{}) ([]byte, error)
	GetTask(refURL string) (*ExecutionResponse, error)
	PollTaskStatus(refURL string, timeout time.Duration) (*ExecutionResponse, error)
	GetPipelineConfig(app, pipelineConfigID string) (*PipelineConfig, error)
	SavePipelineConfig(pipelineConfig PipelineConfig) error
	ListPipelineConfigs(app string) ([]PipelineConfig, error)
	DeletePipeline(app, pipelineConfigID string) error
	FiatLogin(fiatUser string, fiatPass string) error
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

func (c *client) applicationTasksURL(app string) string {
	return c.endpoint + fmt.Sprintf("/applications/%s/tasks", app)
}

func (c *client) applicationURL(app string) string {
	return c.endpoint + fmt.Sprintf("/applications/%s", app)
}

func (c *client) applicationsURL() string {
	return c.endpoint + "/applications"
}

func (c *client) pipelineURL(app string, pipelineID string) string {
	return fmt.Sprintf("%s/pipelines/%s/%s", c.endpoint, app, pipelineID)
}

func (c *client) fiatLoginURL() string {
	return c.endpoint + "/login"
}

func (c *client) templateExists(id string) (bool, error) {
	url := c.pipelineTemplatesURL() + "/" + id
	resp, _, err := c.getJSON(url)

	if err != nil {
		return false, err
	}

	logrus.WithField("status", resp.StatusCode).Debug("Response")

	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}
	if resp.StatusCode == http.StatusOK {
		return true, nil
	}
	return false, errors.New("Unable to determine state of the pipeline template " + id + ", status: " + strconv.Itoa(resp.StatusCode))
}

// PublishTemplateOptions options for publishing templates
type PublishTemplateOptions struct {
	SkipPlan   bool
	TemplateID string
	Source     string
}

func (c *client) PublishTemplate(template map[string]interface{}, options PublishTemplateOptions) (*TaskRefResponse, error) {
	url := c.pipelineTemplatesURL()
	if options.TemplateID != "" {
		// add the ability to override the template ID when publishing
		template["id"] = options.TemplateID
	}
	if options.Source != "" {
		// add the ability to override the source template when publishing
		template["source"] = options.Source
	}
	id := template["id"].(string)
	exists, err := c.templateExists(id)
	if err != nil {
		return nil, errors.Wrap(err, "unable to check status of template")
	}
	if exists {
		url = url + "/" + id
	}
	if options.SkipPlan {
		url = url + "?skipPlanDependents=true"
	}
	resp, respBody, err := c.postJSON(url, template)

	if err != nil {
		return nil, errors.Wrap(err, "pipeline template publish")
	}

	logrus.WithFields(logrus.Fields{
		"status": resp.StatusCode,
		"body":   string(respBody),
	}).Debug("Response")

	if resp.StatusCode != http.StatusAccepted {
		return nil, errors.New("create template request failed")
	}

	var ref TaskRefResponse
	if err := json.Unmarshal(respBody, &ref); err != nil {
		return nil, errors.New("unmarshaling create template response")
	}

	return &ref, nil
}

func (c *client) ApplicationSubmitTask(app string, task Task) (*TaskRefResponse, error) {
	url := c.applicationTasksURL(app)
	resp, respBody, err := c.postJSON(url, task)

	if err != nil {
		return nil, errors.Wrap(err, "create application submit task")
	}

	logrus.WithFields(logrus.Fields{
		"status": resp.StatusCode,
		"body":   string(respBody),
	}).Debug("Response")

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("submit task failed")
	}

	var ref TaskRefResponse
	if err := json.Unmarshal(respBody, &ref); err != nil {
		return nil, errors.New("unmarshaling task create response")
	}

	return &ref, nil
}

func (c *client) ApplicationGet(app string) (bool, []byte, error) {
	url := c.applicationURL(app)
	resp, respBody, err := c.getJSON(url)

	if err != nil {
		return false, nil, errors.Wrap(err, "unable to get application info")
	}

	logrus.WithFields(logrus.Fields{
		"status": resp.StatusCode,
		"body":   string(respBody),
	}).Debug("Response")

	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusForbidden {
		return false, nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return false, nil, errors.New("Unable to determine state of application " + app + ", status: " + strconv.Itoa(resp.StatusCode))
	}

	return true, respBody, nil
}

func (c *client) ApplicationList() ([]ApplicationInfo, error) {
	url := c.applicationsURL()
	resp, respBody, err := c.getJSON(url)

	if err != nil {
		return nil, errors.Wrap(err, "unable to get application list")
	}

	logrus.WithFields(logrus.Fields{
		"status": resp.StatusCode,
		"body":   string(respBody),
	}).Debug("Response")

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("Unable to fetch application list: " + strconv.Itoa(resp.StatusCode))
	}

	var appInfo []ApplicationInfo
	if err := json.Unmarshal(respBody, &appInfo); err != nil {
		return nil, errors.New("unmarshaling application list")
	}

	return appInfo, nil
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

	logrus.WithFields(logrus.Fields{
		"status": resp.StatusCode,
		"body":   string(respBody),
	}).Debug("Response")

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusBadRequest {
			return respBody, ErrInvalidPipelineTemplate
		}
		return respBody, errors.New("plan request failed")
	}

	return respBody, nil
}

func (c *client) DeleteTemplate(templateID string) (*TaskRefResponse, error) {
	url := c.pipelineTemplatesURL() + "/" + templateID
	resp, respBody, err := c.delete(url)

	if err != nil {
		return nil, errors.Wrap(err, "delete request failed")
	}

	logrus.WithFields(logrus.Fields{
		"status": resp.StatusCode,
		"body":   string(respBody),
	}).Debug("Response")

	if resp.StatusCode != http.StatusAccepted {
		return nil, errors.New("delete request failed")
	}

	var ref TaskRefResponse
	if err := json.Unmarshal(respBody, &ref); err != nil {
		return nil, errors.New("failed to unmarshall delete template response")
	}

	return &ref, nil
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

	logrus.WithFields(logrus.Fields{
		"status": resp.StatusCode,
		"body":   string(respBody),
	}).Debug("Response")

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("get task status failed")
	}

	var task ExecutionResponse
	if err := json.Unmarshal(respBody, &task); err != nil {
		return nil, errors.Wrap(err, "failed unmarshaling task status response")
	}

	return &task, nil
}

func (c *client) PollTaskStatus(refURL string, timeout time.Duration) (*ExecutionResponse, error) {
	logrus.WithField("refURL", refURL).Info("Waiting for task to complete...")

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

	logrus.WithFields(logrus.Fields{
		"status": resp.StatusCode,
		"body":   string(respBody),
	}).Debug("Response")

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return nil, nil
		}
		return nil, errors.New("get pipeline config failed")
	}

	// TODO rz - HACK: Spinnaker bug returning 200 on a pipeline config that isn't found
	if len(respBody) == 0 {
		return nil, nil
	}

	var config PipelineConfig
	if err := json.Unmarshal(respBody, &config); err != nil {
		return nil, errors.Wrap(err, "failed unmarshaling pipeline config response")
	}

	return &config, nil
}

func (c *client) ListPipelineConfigs(app string) ([]PipelineConfig, error) {
	url := c.pipelineConfigsURL(app)
	resp, respBody, err := c.getJSON(url)

	if err != nil {
		return nil, errors.Wrap(err, "unable to get pipeline list")
	}

	logrus.WithFields(logrus.Fields{
		"status": resp.StatusCode,
		"body":   string(respBody),
	}).Debug("Response")

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("Unable to fetch pipeline list: " + strconv.Itoa(resp.StatusCode))
	}

	var pipelineInfo []PipelineConfig
	if err := json.Unmarshal(respBody, &pipelineInfo); err != nil {
		return nil, errors.New("unmarshaling pipeline list")
	}

	return pipelineInfo, nil
}

func (c *client) SavePipelineConfig(pipelineConfig PipelineConfig) error {
	url := c.pipelinesURL()
	logrus.WithField("url", url).Debug("saving pipeline")
	resp, respBody, err := c.postJSON(url, pipelineConfig)

	if err != nil {
		return errors.Wrap(err, "save pipeline config")
	}

	logrus.WithFields(logrus.Fields{
		"status": resp.StatusCode,
		"body":   string(respBody),
	}).Debug("Response")

	if resp.StatusCode != http.StatusOK {
		return errors.New("plan request failed")
	}

	return nil
}

func (c *client) DeletePipeline(app string, pipelineID string) error {
	url := c.pipelineURL(app, pipelineID)
	logrus.WithField("pipelineConfigID", pipelineID).Debug("deleting pipeline")

	resp, respBody, err := c.delete(url)

	if err != nil {
		return errors.Wrap(err, "delete pipeline config")
	}

	logrus.WithFields(logrus.Fields{
		"status": resp.StatusCode,
		"body":   string(respBody),
	}).Debug("Response")

	if resp.StatusCode != http.StatusOK {
		return errors.New("delete request failed")
	}

	return nil
}

func (c *client) FiatLogin(fiatUser string, fiatPass string) error {
	postURL := c.fiatLoginURL()

	data := url.Values{"username": {fiatUser}, "password": {fiatPass}, "submit": {"Login"}}

	_, _, err := c.postForm(postURL, data)
	if err != nil {
		return errors.Wrap(err, "fiat login")
	}

	return nil
}
