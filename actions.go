package roer

import (
	"encoding/json"
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

func AppCreateAction(clientConfig spinnaker.ClientConfig) cli.ActionFunc {
	return func(cc *cli.Context) error {
		appName := cc.Args().Get(0)
		ownerEmail := cc.Args().Get(1)
		logrus.WithField("appName", appName).Debug("Filling in create application task")

		createAppJob := spinnaker.CreateApplicationJob{
			Application: spinnaker.ApplicationAttributes{
				Email: ownerEmail,
				Name:  appName,
			},
			Type: "createApplication",
		}
		createApp := spinnaker.Task{
			Application: appName,
			Description: "Create Application: " + appName,
			Job:         []interface{}{createAppJob},
		}

		client, err := clientFromContext(cc, clientConfig)
		if err != nil {
			return errors.Wrapf(err, "creating spinnaker client")
		}

		logrus.Info("Sending create app task")
		ref, err := client.ApplicationSubmitTask(appName, createApp)
		if err != nil {
			return errors.Wrapf(err, "submitting task")
		}

		resp, err := client.PollTaskStatus(ref.Ref, 1*time.Minute)
		if err != nil {
			return errors.Wrap(err, "poll create app status")
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

func AppGetAction(clientConfig spinnaker.ClientConfig) cli.ActionFunc {
	return func(cc *cli.Context) error {
		appName := cc.Args().Get(0)

		client, err := clientFromContext(cc, clientConfig)
		if err != nil {
			return errors.Wrapf(err, "creating spinnaker client")
		}

		logrus.WithField("appName", appName).Info("Fetching application")
		exists, appInfo, err := client.ApplicationGet(appName)
		if err != nil {
			return errors.Wrap(err, "Fetching app info")
		}

		if exists == false {
			fmt.Println("App does not exist or insufficient permission")
			return fmt.Errorf("Could not fetch app info")
		}
		prettyPrintJSON(appInfo)
		return nil
	}
}

func AppListAction(clientConfig spinnaker.ClientConfig) cli.ActionFunc {
	return func(cc *cli.Context) error {
		client, err := clientFromContext(cc, clientConfig)
		if err != nil {
			return errors.Wrapf(err, "creating spinnaker client")
		}

		logrus.Info("Fetching application list")

		appInfo, err := client.ApplicationList()
		if err != nil {
			return errors.Wrap(err, "Fetching application list")
		}

		for _, app := range appInfo {
			fmt.Println(app.Name)
		}

		return nil
	}
}

// Save a pipeline from json source
func PipelineSaveJsonAction(clientConfig spinnaker.ClientConfig) cli.ActionFunc {
	return func(cc *cli.Context) error {
		jsonFile := cc.Args().Get(0)
		logrus.WithField("file", jsonFile).Debug("Reading JSON payload")
		dat, err := ioutil.ReadFile(jsonFile)
		if err != nil {
			return errors.Wrapf(err, "reading JSON file: %s", jsonFile)
		}

		client, err := clientFromContext(cc, clientConfig)
		if err != nil {
			return errors.Wrapf(err, "creating spinnaker client")
		}

		var newConfig spinnaker.PipelineConfig
		if err := json.Unmarshal(dat, &newConfig); err != nil {
			return errors.Wrap(err, "Unmarshaling JSON pipeline")
		}

		existingConfig, err := client.GetPipelineConfig(newConfig.Application, newConfig.Name)
		if err != nil {
			return errors.Wrap(err, "seaching for existing pipeline config")
		}

		if existingConfig != nil {
			newConfig.ID = existingConfig.ID
		}

		if err := client.SavePipelineConfig(newConfig); err != nil {
			return errors.Wrap(err, "saving pipeline config")
		}

		return nil
	}
}

func PipelineListAction(clientConfig spinnaker.ClientConfig) cli.ActionFunc {
	return func(cc *cli.Context) error {
		appName := cc.Args().Get(0)
		logrus.WithField("app", appName).Debug("Fetching pipelines")

		client, err := clientFromContext(cc, clientConfig)
		if err != nil {
			return errors.Wrapf(err, "creating spinnaker client")
		}

		pipelineInfo, err := client.ListPipeline(appName)
		if err != nil {
			return errors.Wrap(err, "Fetching pipelines")
		}

		for _, pipeline := range pipelineInfo {
			fmt.Println(pipeline.Name)
		}
		return nil
	}
}

func PipelineGetAction(clientConfig spinnaker.ClientConfig) cli.ActionFunc {
	return func(cc *cli.Context) error {
		appName := cc.Args().Get(0)
		pipelineName := cc.Args().Get(1)
		logrus.WithField("app", appName).WithField("pipelineName", pipelineName).Debug("Fetching pipeline")

		client, err := clientFromContext(cc, clientConfig)
		if err != nil {
			return errors.Wrapf(err, "creating spinnaker client")
		}

		pipelineConfig, err := client.GetPipelineConfig(appName, pipelineName)
		if err != nil {
			return errors.Wrap(err, "Fetching pipeline")
		}

		jsonStr, _ := json.Marshal(pipelineConfig)
		fmt.Println(string(jsonStr))
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
		ref, err := client.PublishTemplate(template, spinnaker.PublishTemplateOptions{
			SkipPlan:   cc.Bool("skipPlan"),
			TemplateID: cc.String("templateId"),
			Source:     cc.String("source"),
		})
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

func PipelineDeleteAction(clientConfig spinnaker.ClientConfig) cli.ActionFunc {
	return func(cc *cli.Context) error {
		app := cc.Args().Get(0)
		pipelineID := cc.Args().Get(1)

		client, err := clientFromContext(cc, clientConfig)
		if err != nil {
			return errors.Wrap(err, "creating spinnaker client")
		}

		logrus.Info("Deleting pipeline")
		err = client.DeletePipeline(app, pipelineID)
		if err != nil {
			return errors.Wrap(err, "deleting pipeline template")
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
