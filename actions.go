package tiller

import (
	"fmt"
	"io/ioutil"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/robzienert/tiller/spinnaker"
	"github.com/urfave/cli"
)

// PipelineTemplatePublishAction creates the ActionFunc for publishing pipeline
// templates.
func PipelineTemplatePublishAction(clientConfig spinnaker.ClientConfig) cli.ActionFunc {
	return func(cc *cli.Context) error {
		templateFile := cc.Args().Get(0)
		logrus.WithField("file", templateFile).Debug("Reading template")
		dat, err := ioutil.ReadFile(templateFile)
		if err != nil {
			return errors.Wrapf(err, "reading template file: %s", templateFile)
		}

		client, err := clientFromContext(cc, clientConfig)
		if err != nil {
			return errors.Wrapf(err, "creating spinnaker client")
		}

		var m map[string]interface{}
		if err := yaml.Unmarshal(dat, &m); err != nil {
			return errors.Wrapf(err, "converting JSON to memory-struct")
		}

		logrus.Info("Publishing template")
		ref, err := client.PublishTemplate(m, cc.Bool("update"))
		if err != nil {
			return errors.Wrap(err, "publishing template")
		}

		resp, err := client.PollTaskStatus(ref.Ref, 1*time.Minute)
		if err != nil {
			return errors.Wrap(err, "polling task status")
		}

		if resp.Status == "TERMINAL" {
			logrus.WithField("status", resp.Status).Error("Task failed")
			if retrofitErr := resp.ExtractRetrofitError(); retrofitErr != nil {
				prettyPrintJSON([]byte(retrofitErr.ResponseBody))
			} else {
				fmt.Printf("%#v\n", resp)
			}
		} else {
			logrus.WithField("status", resp.Status).Info("Task completed")
		}

		return nil
	}
}

// PipelineTemplatePlanAction creates the ActionFunc for planning a pipeline
// template with a given configuration.
func PipelineTemplatePlanAction(clientConfig spinnaker.ClientConfig) cli.ActionFunc {
	return func(cc *cli.Context) error {
		configFile := cc.Args().Get(0)

		logrus.WithField("file", configFile).Debug("Reading config")
		dat, err := ioutil.ReadFile(configFile)
		if err != nil {
			return errors.Wrapf(err, "reading configuration file: %s", configFile)
		}

		client, err := clientFromContext(cc, clientConfig)
		if err != nil {
			return errors.Wrapf(err, "creating spinnaker client")
		}

		var m map[string]interface{}
		if err := yaml.Unmarshal(dat, &m); err != nil {
			return errors.Wrapf(err, "converting JSON to memory-struct")
		}

		resp, err := client.Plan(m)
		if err != nil {
			if err == spinnaker.ErrInvalidPipelineTemplate {
				prettyPrintJSON(resp)
				return nil
			}
			fmt.Println(resp)
			return errors.Wrap(err, "planning configuration")
		}

		prettyPrintJSON(resp)
		return nil
	}
}

func clientFromContext(cc *cli.Context, config spinnaker.ClientConfig) (spinnaker.Client, error) {
	hc, err := config.HTTPClientFactory(cc)
	if err != nil {
		return nil, errors.Wrap(err, "creating http client from context")
	}
	return spinnaker.New(config.Endpoint, hc), nil
}
