package roer

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/ghodss/yaml"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spinnaker/roer/spinnaker"
	"gopkg.in/urfave/cli.v1"
	"regexp"
	"strings"
)

// PipelineExecAction requests a pipeline execution and optionally waits for
// it to complete. Arguments are the name of the app and the name of the
// pipeline to start.
func PipelineExecAction(clientConfig spinnaker.ClientConfig) cli.ActionFunc {
	return func(cc *cli.Context) error {
		appName := cc.Args().Get(0)
		pipelineName := cc.Args().Get(1)
		monitor := cc.Bool("monitor")
		numRetries := cc.Int("retry")

		logrus.WithFields(logrus.Fields{
			"app":      appName,
			"pipeline": pipelineName,
			"monitor":  monitor,
			"retries":  numRetries,
		}).Info("Executing Pipeline...")

		client, err := clientFromContext(cc, clientConfig)
		if err != nil {
			return errors.Wrapf(err, "creating spinnaker client")
		}

		resp, err := client.ExecPipeline(appName, pipelineName)
		if err != nil {
			return errors.Wrapf(err, "couldn't execute pipeline")
		}
		logrus.Infof("Ref task id: %s", resp.Ref)
		if monitor {
			var err error
			var execResp *spinnaker.ExecutionResponse
			for retryCounter := 0; retryCounter <= numRetries; {
				retryCounter++
				logrus.Infof("Polling tasks status, retry number: %d", retryCounter)
				execResp, err = client.PollTaskStatus(resp.Ref, 30*time.Minute)
				if err != nil {
					logrus.WithField("exec_response", execResp).Errorf("Executing response error: %v", err)
				}
			}
			if err != nil {
				return err
			}
			if execResp != nil && execResp.Status != "SUCCEEDED" {
				return fmt.Errorf("pipeline did not complete with a SUCCESS status.  Ended with status: %s", execResp.Status)
			}
		}
		return nil
	}
}

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
			return errors.Wrap(err, "searching for existing pipeline config")
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

// AppCreateAction creates the ActionFunc for creating a spinnaker application
func AppCreateAction(clientConfig spinnaker.ClientConfig) cli.ActionFunc {
	return func(cc *cli.Context) error {
		appName := cc.Args().Get(0)
		configFile := cc.Args().Get(1)

		logrus.WithField("appName", appName).Debug("Filling in create application task")
		logrus.WithField("file", configFile).Debug("Reading application config")

		config, err := readYamlFile(configFile)
		if err != nil {
			return errors.Wrapf(err, "reading config file: %s", configFile)
		}

		config["name"] = appName

		createAppJob := spinnaker.ApplicationJob{
			Application: config,
			Type:        "createApplication",
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

		resp, err := client.PollTaskStatus(ref.Ref, time.Duration(cc.GlobalInt("timeout"))*time.Second)
		if err != nil {
			return errors.Wrap(err, "poll create app status")
		}

		if resp.Status == "TERMINAL" {
			logrus.WithField("status", resp.Status).Error("Task failed")
			if retrofitErr := resp.ExtractRetrofitError(); retrofitErr != nil {
				prettyPrintJSON([]byte(retrofitErr.ResponseBody))
			} else {
				logrus.Debugf("Response data %#v", resp)
			}
		} else {
			logrus.WithField("status", resp.Status).Info("Task completed")
		}

		return nil
	}
}

// AppDeleteAction delete an application
func AppDeleteAction(clientConfig spinnaker.ClientConfig) cli.ActionFunc {
	return func(cc *cli.Context) error {
		appName := cc.Args().Get(0)

		config := make(map[string]interface{})

		config["name"] = appName

		deleteAppJob := spinnaker.ApplicationJob{
			Application: config,
			Type:        "deleteApplication",
		}

		deleteApp := spinnaker.Task{
			Application: appName,
			Description: "Delete Application: " + appName,
			Job:         []interface{}{deleteAppJob},
		}

		client, err := clientFromContext(cc, clientConfig)
		if err != nil {
			return errors.Wrapf(err, "creating spinnaker client")
		}

		logrus.Info("Sending delete app task")
		ref, err := client.ApplicationSubmitTask(appName, deleteApp)
		if err != nil {
			return errors.Wrapf(err, "submitting task")
		}

		resp, err := client.PollTaskStatus(ref.Ref, time.Duration(cc.GlobalInt("timeout"))*time.Second)
		if err != nil {
			return errors.Wrap(err, "poll delete app status")
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

// AppGetAction creates the ActionFunc for fetching spinnaker application configuration
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
			logrus.Error("App does not exist or insufficient permission")
			return fmt.Errorf("Could not fetch app info")
		}
		prettyPrintJSON(appInfo)
		return nil
	}
}

// AppListAction creates the ActionFunc for listing applications
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
			logrus.Debug(app.Name)
		}

		return nil
	}
}

func AppHistoryAction(clientConfig spinnaker.ClientConfig) cli.ActionFunc {
	return func(cc *cli.Context) error {
		appName := cc.Args().Get(0)

		client, err := clientFromContext(cc, clientConfig)
		if err != nil {
			return errors.Wrapf(err, "creating spinnaker client")
		}

		history, err := client.ApplicationHistory(appName)
		if err != nil {
			return errors.Wrap(err, "Fetching application history")
		}

		pipelineName := cc.String("pipeline")
		for _, e := range history {
			if pipelineName == "" || pipelineName == e.Name {
				fmt.Printf("%d %s %s %s\n", e.StartTime, e.ID, e.Status, e.Name)
			}
		}

		return nil
	}
}

func ExecGetAction(clientConfig spinnaker.ClientConfig) cli.ActionFunc {
	return func(cc *cli.Context) error {
		execID := cc.Args().Get(0)

		client, err := clientFromContext(cc, clientConfig)
		if err != nil {
			return errors.Wrap(err, "creating spinnaker client")
		}

		exec, err := client.Execution(execID)
		if err != nil {
			return errors.Wrap(err, "Fetching execution")
		}

		prettyPrintJSON(exec)
		return nil
	}
}

// Save a pipeline from json source
func PipelineSaveJSONAction(clientConfig spinnaker.ClientConfig) cli.ActionFunc {
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
			return errors.Wrap(err, "searching for existing pipeline config")
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

// PipelineListConfigsAction creates the ActionFunc for listing pipeline configs
func PipelineListConfigsAction(clientConfig spinnaker.ClientConfig) cli.ActionFunc {
	return func(cc *cli.Context) error {
		appName := cc.Args().Get(0)
		logrus.WithField("app", appName).Debug("Fetching pipelines")

		client, err := clientFromContext(cc, clientConfig)
		if err != nil {
			return errors.Wrapf(err, "creating spinnaker client")
		}

		pipelineInfo, err := client.ListPipelineConfigs(appName)
		if err != nil {
			return errors.Wrap(err, "Fetching pipelines")
		}

		for _, pipeline := range pipelineInfo {
			logrus.Debug(pipeline.Name)
		}
		return nil
	}
}

// PipelineGetConfigAction creates the ActionFunc for fetching a pipeline config
func PipelineGetConfigAction(clientConfig spinnaker.ClientConfig) cli.ActionFunc {
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
		logrus.Debug(string(jsonStr))
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

		resp, err := client.PollTaskStatus(ref.Ref, time.Duration(cc.GlobalInt("timeout"))*time.Second)
		if err != nil {
			return errors.Wrap(err, "polling task status")
		}

		if resp.Status == "TERMINAL" {
			logrus.WithField("status", resp.Status).Error("Task failed")
			if retrofitErr := resp.ExtractRetrofitError(); retrofitErr != nil {
				prettyPrintJSON([]byte(retrofitErr.ResponseBody))
			} else {
				logrus.Debugf("Response data %#v", resp)
			}
		} else {
			logrus.WithField("status", resp.Status).Info("Task completed")
		}

		return nil
	}
}

// PipelineTemplateRenderAction creates a rendered template and writes it to disk
func PipelineTemplateRenderAction(clientConfig spinnaker.ClientConfig) cli.ActionFunc {
	return func(cc *cli.Context) error {
		templateFile := cc.Args().Get(0)

		template, err := ioutil.ReadFile(templateFile)
		if err != nil {
			return errors.Wrapf(err, "reading template file: %s", templateFile)
		}

		templateStr := string(template)

		valuesFile := cc.Args().Get(1)
		valuesData, err := ioutil.ReadFile(valuesFile)
		if err != nil {
			return errors.Wrapf(err, "reading values file: %s", valuesFile)
		}

		var values map[string]string
		if err := json.Unmarshal(valuesData, &values); err != nil {
			return errors.Wrapf(err, "unmarshaling json in %s", valuesFile)
		}

		var tmplVarsRegex = regexp.MustCompile(`{{ *([a-zA-Z0-9_.-]*) *}}`)
		submatches := tmplVarsRegex.FindAllStringSubmatch(templateStr, -1)
		for _, submatch := range submatches {
			fullMatch := submatch[0]
			keyMatch := submatch[1]
			substitueVal := values[keyMatch]
			templateStr = strings.Replace(templateStr, fullMatch, substitueVal, -1)
		}

		outputFile := cc.Args().Get(2)
		fmt.Println("Outputting rendered template to:", outputFile)
		ioutil.WriteFile(outputFile, []byte(templateStr), 0644)
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
			logrus.Debug(string(resp))
			return errors.Wrap(err, "planning configuration")
		}

		prettyPrintJSON(resp)
		return nil
	}
}

// PipelineTemplateConvertAction creates the ActionFunc for converting an existing pipeline
// into a pipeline template
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
			logrus.Debug(resp)
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

		logrus.Debug(generatedTemplateHeader)
		logrus.Debug(string(template))

		return nil
	}
}

// PipelineTemplateDeleteAction creates the ActionFunc for deleting a pipeline template
func PipelineTemplateDeleteAction(clientConfig spinnaker.ClientConfig) cli.ActionFunc {
	return func(cc *cli.Context) error {
		pipelineTemplateID := cc.Args().Get(0)

		client, err := clientFromContext(cc, clientConfig)
		if err != nil {
			return errors.Wrap(err, "creating spinnaker client")
		}

		logrus.Info("Deleting template")
		ref, err := client.DeleteTemplate(pipelineTemplateID)
		if err != nil {
			return errors.Wrap(err, "deleting pipeline template")
		}

		resp, err := client.PollTaskStatus(ref.Ref, time.Duration(cc.GlobalInt("timeout"))*time.Second)
		if err != nil {
			return errors.Wrap(err, "polling task status")
		}

		if resp.Status == "TERMINAL" {
			logrus.WithField("status", resp.Status).Error("Task failed")
			if retrofitErr := resp.ExtractRetrofitError(); retrofitErr != nil {
				prettyPrintJSON([]byte(retrofitErr.ResponseBody))
			} else {
				logrus.Debugf("Response data %#v", resp)
			}
		} else {
			logrus.WithField("status", resp.Status).Info("Task completed")
		}

		return nil
	}
}

// PipelineDeleteAction creates the ActionFunc for deleting a pipeline
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

	var sc spinnaker.Client
	sc = spinnaker.New(config.Endpoint, hc)

	if cc.GlobalIsSet("fiatUser") && cc.GlobalIsSet("fiatPass") {
		err := sc.FiatLogin(cc.GlobalString("fiatUser"), cc.GlobalString("fiatPass"))
		if err != nil {
			return nil, errors.Wrap(err, "fiat auth login attempt")
		}
	}

	return sc, nil
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
