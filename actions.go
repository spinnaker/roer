package roer

import (
	"fmt"
	"io/ioutil"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/ghodss/yaml"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/spinnaker/roer/spinnaker"
	"github.com/urfave/cli"
)

// PipelineSaveAction creates the ActionFunc for saving pipeline configurations.
func PipelineSaveAction(clientConfig spinnaker.ClientConfig) cli.ActionFunc {
	return func(cc *cli.Context) error {
		configFile := cc.Args().Get(0)
		logrus.WithField("file", configFile).Debug("Reading config")
		dat, err := ioutil.ReadFile(configFile)
		if err != nil {
			return errors.Wrapf(err, "reading config file: %s", configFile)
		}

		client, err := clientFromContext(cc, clientConfig)
		if err != nil {
			return errors.Wrapf(err, "creating spinnaker client")
		}

		var m map[string]interface{}
		if err := yaml.Unmarshal(dat, &m); err != nil {
			return errors.Wrapf(err, "unmarshaling config")
		}

		if _, ok := m["schema"]; !ok {
			logrus.Error("Pipeline save command currently only supports pipeline template configurations")
		}

		var config PipelineConfiguration
		if err := mapstructure.Decode(m, &config); err != nil {
			return errors.Wrap(err, "converting map to struct")
		}

		existingConfig, err := client.GetPipelineConfig(config.Pipeline.Application, config.Pipeline.Name)
		if err != nil {
			return errors.Wrap(err, "seaching for existing pipeline config")
		}

		// TODO rz - orca should probably auto-set the pipeline config id somehow so
		// executions correctly show up in the UI.
		payload := config.ToClient()
		if existingConfig != nil {
			payload.ID = existingConfig.ID
		}

		if err := client.SavePipelineConfig(payload); err != nil {
			return errors.Wrap(err, "saving pipeline config")
		}

		return nil
	}
}

// PipelineTemplatePublishAction creates the ActionFunc for publishing pipeline
// templates.
func PipelineTemplatePublishAction(clientConfig spinnaker.ClientConfig) cli.ActionFunc {
	return func(cc *cli.Context) error {
		if cc.Bool("update") {
			logrus.Warn("The `update` flag is deprecated, `publish` always creates or updates the template")
		}
		templateFile := cc.Args().Get(0)
		logrus.WithField("file", templateFile).Debug("Reading template")

		template, err := readYamlFile(templateFile)
		if err != nil {
			return errors.Wrapf(err, "reading template file: %s", templateFile)
		}

		client, err := clientFromContext(cc, clientConfig)
		if err != nil {
			return errors.Wrapf(err, "creating spinnaker client")
		}

		logrus.Info("Publishing template")
		ref, err := client.PublishTemplate(template, cc.Bool("skipPlan"))
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
		config, err := readYamlFile(configFile)

		var template map[string]interface{}
		if cc.IsSet("template") {
			logrus.WithField("file", cc.String("template")).Debug("Reading template")
			template, err = readYamlFile(cc.String("template"))
		}

		client, err := clientFromContext(cc, clientConfig)
		if err != nil {
			return errors.Wrapf(err, "creating spinnaker client")
		}

		resp, err := client.Plan(config, template)
		if err != nil {
			if err == spinnaker.ErrInvalidPipelineTemplate {
				prettyPrintJSON(resp)
				return nil
			}
			fmt.Println(string(resp))
			return errors.Wrap(err, "planning configuration")
		}

		prettyPrintJSON(resp)
		return nil
	}
}

func PipelineTemplateConvertAction(clientConfig spinnaker.ClientConfig) cli.ActionFunc {
	return func(cc *cli.Context) error {
		app := cc.Args().Get(0)
		pipelineConfigID := cc.Args().Get(1)

		client, err := clientFromContext(cc, clientConfig)
		if err != nil {
			return errors.Wrap(err, "creating spinnaker client")
		}

		resp, err := client.GetPipelineConfig(app, pipelineConfigID)
		if err != nil {
			fmt.Println(resp)
			return errors.Wrap(err, "getting pipeline config")
		}

		if resp == nil {
			logrus.Error("could not find pipeline config")
		}

		// TODO rz - Write custom marshaler to preserve key order
		template, err := yaml.Marshal(convertPipelineToTemplate(*resp))
		if err != nil {
			return errors.Wrap(err, "marshaling template to YAML")
		}

		fmt.Println(generatedTemplateHeader)
		fmt.Println(string(template))

		return nil
	}
}

func PipelineTemplateDeleteAction(clientConfig spinnaker.ClientConfig) cli.ActionFunc {
	return func(cc *cli.Context) error {
		pipelineTemplateId := cc.Args().Get(0)

		client, err := clientFromContext(cc, clientConfig)
		if err != nil {
			return errors.Wrap(err, "creating spinnaker client")
		}

		logrus.Info("Deleting template")
		ref, err := client.DeleteTemplate(pipelineTemplateId)
		if err != nil {
			return errors.Wrap(err, "deleting pipeline template")
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

func clientFromContext(cc *cli.Context, config spinnaker.ClientConfig) (spinnaker.Client, error) {
	hc, err := config.HTTPClientFactory(cc)
	if err != nil {
		return nil, errors.Wrap(err, "creating http client from context")
	}
	return spinnaker.New(config.Endpoint, hc), nil
}

func readYamlFile(f string) (map[string]interface{}, error) {
	configDat, err := ioutil.ReadFile(f)
	if err != nil {
		return nil, errors.Wrapf(err, "reading file: %s", f)
	}

	var m map[string]interface{}
	if err := yaml.Unmarshal(configDat, &m); err != nil {
		return nil, errors.Wrapf(err, "unmarshaling yaml in %s", f)
	}

	return m, nil
}
