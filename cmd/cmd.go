package cmd

import (
	"errors"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spinnaker/roer"
	"github.com/spinnaker/roer/spinnaker"
	"gopkg.in/urfave/cli.v1"
)

// NewRoer returns a new instance of the OSS roer application
func NewRoer(version string, clientConfig spinnaker.ClientConfig) *cli.App {
	cli.VersionFlag = cli.BoolFlag{Name: "version"}
	cli.HelpFlag = cli.BoolFlag{Name: "help", Usage: "Show Help", Hidden: true}

	app := cli.NewApp()
	app.Name = "roer"
	app.Usage = "Spinnaker CLI"
	app.Version = version
	app.Commands = []cli.Command{
		{
			Name:  "pipeline",
			Usage: "pipeline tasks",
			Subcommands: []cli.Command{
				{
					Name:      "save",
					Usage:     "save a pipeline configuration",
					ArgsUsage: "[configuration.yml]",
					Before: func(cc *cli.Context) error {
						if cc.NArg() != 1 {
							return errors.New("path to configuration file is required")
						}
						return nil
					},
					Action: roer.PipelineSaveAction(clientConfig),
				},
				{
					Name:      "savejson",
					Usage:     "save a json pipeline configuration",
					ArgsUsage: "[configuration.json]",
					Before: func(cc *cli.Context) error {
						if cc.NArg() != 1 {
							return errors.New("path to json file is required")
						}
						return nil
					},
					Action: roer.PipelineSaveJSONAction(clientConfig),
				},
				{
					Name:      "list",
					Usage:     "list all the pipelines in an application",
					ArgsUsage: "[application name]",
					Before: func(cc *cli.Context) error {
						if cc.NArg() != 1 {
							return errors.New("name of application is required")
						}
						return nil
					},
					Action: roer.PipelineListConfigsAction(clientConfig),
				},
				{
					Name:      "get",
					Usage:     "get the config for an individual pipeline",
					ArgsUsage: "[application name] [pipeline name]",
					Before: func(cc *cli.Context) error {
						if cc.NArg() != 2 {
							return errors.New("both app name and pipeline name are required")
						}
						return nil
					},
					Action: roer.PipelineGetConfigAction(clientConfig),
				},
				{
					Name:      "delete",
					Usage:     "delete a pipeline",
					ArgsUsage: "[application] [pipelineName]",
					Before: func(cc *cli.Context) error {
						if cc.NArg() != 2 {
							return errors.New("requires an application and a pipeline")
						}
						return nil
					},
					Action: roer.PipelineDeleteAction(clientConfig),
				},
			},
		},
		{
			Name:  "app",
			Usage: "application tasks",
			Subcommands: []cli.Command{
				{
					Name:      "create",
					Usage:     "create or update an application",
					ArgsUsage: "[app name] [config.yml]",
					Before: func(cc *cli.Context) error {
						if cc.NArg() != 2 {
							return errors.New("both application name and configuration are required")
						}
						return nil
					},
					Action: roer.AppCreateAction(clientConfig),
				},
				{
					Name:      "delete",
					Usage:     "delete an application",
					ArgsUsage: "[app name]",
					Before: func(cc *cli.Context) error {
						if cc.NArg() != 1 {
							return errors.New("application name is required")
						}
						return nil
					},
					Action: roer.AppDeleteAction(clientConfig),
				},
				{
					Name:      "get",
					Usage:     "get info about an application",
					ArgsUsage: "[app name]",
					Before: func(cc *cli.Context) error {
						if cc.NArg() != 1 {
							return errors.New("application name is required")
						}
						return nil
					},
					Action: roer.AppGetAction(clientConfig),
				},
				{
					Name:   "list",
					Usage:  "list applications",
					Action: roer.AppListAction(clientConfig),
				},
				{
					Name:      "exec",
					Usage:     "execute pipeline",
					ArgsUsage: "[application name] [pipeline name]",
					Flags: []cli.Flag{
						cli.BoolFlag{
							Name:  "monitor, m",
							Usage: "Continue to monitor the executing of the pipeline",
						},
						cli.IntFlag{
							Name:  "retry, r",
							Usage: "Number of times to have the monitor retry if a call fails or times out",
						},
					},
					Before: func(cc *cli.Context) error {
						if cc.NArg() != 2 {
							return errors.New("app name and pipeline are required")
						}
						return nil
					},
					Action: roer.PipelineExecAction(clientConfig),
				},
				{
					Name:      "history",
					Usage:     "list pipeline executions",
					ArgsUsage: "[app name]",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "pipeline, p",
							Usage: "filter return to only the named pipeline",
						},
					},
					Before: func(cc *cli.Context) error {
						if cc.NArg() != 1 {
							return errors.New("application name is required")
						}
						return nil
					},
					Action: roer.AppHistoryAction(clientConfig),
				},
			},
		},
		{
			Name:  "execution",
			Usage: "commands that work with a single execution",
			Subcommands: []cli.Command{
				{
					Name:      "get",
					Usage:     "returns the execution json",
					ArgsUsage: "[exec id]",
					Before: func(cc *cli.Context) error {
						if cc.NArg() != 1 {
							return errors.New("execution ID is required")
						}
						return nil
					},
					Action: roer.ExecGetAction(clientConfig),
				},
			},
		},
		{
			Name:  "pipeline-template",
			Usage: "pipeline template tasks",
			Subcommands: []cli.Command{
				{
					Name:      "publish",
					Usage:     "publish a pipeline template, will create or update a template",
					ArgsUsage: "[template.yml]",
					Flags: []cli.Flag{
						cli.BoolFlag{
							Name:  "update, u",
							Usage: "DEPRECATED: update the given pipeline, the default action always creates or updates",
						},
						cli.BoolFlag{
							Name:  "skipPlan, s",
							Usage: "skip the plan dependent pipelines safety feature",
						},
						cli.StringFlag{
							Name:  "templateId, t",
							Usage: "override the template ID",
						},
						cli.StringFlag{
							Name:  "source",
							Usage: "override or add the source template",
						},
					},
					Before: func(cc *cli.Context) error {
						if cc.NArg() != 1 {
							return errors.New("path to template file is required")
						}
						return nil
					},
					Action: roer.PipelineTemplatePublishAction(clientConfig),
				},
				{
					Name:  "render",
					Usage: "render a pipeline json based on a template using string substitution",
					Description: `
		Given a pipeline template render a pipeline JSON locally. A json file with
		values is substituted to keys in the given JSON template
					`,
					ArgsUsage: "[template.json] [values.json] [output.json]",
					Before: func(cc *cli.Context) error {
						if cc.NArg() != 3 {
							return errors.New("path to template.json, values.json and output.json are required")
						}
						return nil
					},
					Action: roer.PipelineTemplateRenderAction(clientConfig),
				},
				{
					Name:  "plan",
					Usage: "validate a pipeline template and or plan a configuration",
					Description: `
		Given a pipeline template configuration, a plan operation
		will be run, with either the errors being returned or the
		final pipeline JSON that would be executed.
					`,
					ArgsUsage: "[configuration.yml]",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "template, t",
							Usage: "local template to inline while planning",
						},
					},
					Before: func(cc *cli.Context) error {
						if cc.NArg() != 1 {
							return errors.New("path to configuration file is required")
						}
						return nil
					},
					Action: roer.PipelineTemplatePlanAction(clientConfig),
				},
				{
					Name:      "convert",
					Usage:     "converts an existing, non-templated pipeline config into a scaffolded template",
					ArgsUsage: "[appName] [pipelineName]",
					Before: func(cc *cli.Context) error {
						if cc.NArg() != 2 {
							return errors.New("appName and pipelineName args are required")
						}
						return nil
					},
					Action: roer.PipelineTemplateConvertAction(clientConfig),
				},
				{
					Name:  "delete",
					Usage: "deletes a pipeline template",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "id",
							Usage: "id of the template to delete",
						},
					},
					Before: func(cc *cli.Context) error {
						if cc.NArg() != 1 {
							return errors.New("id is required")
						}
						return nil
					},
					Action: roer.PipelineTemplateDeleteAction(clientConfig),
				},
				// {
				// 	Name:      "run",
				// 	Usage:     "run a pipeline",
				// 	ArgsUsage: "[configuration.yml]",
				// 	Before: func(cc *cli.Context) error {
				// 		if cc.NArg() != 1 {
				// 			return errors.New("path to configuration file is required")
				// 		}
				// 		return nil
				// 	},
				// 	Action: roer.PipelineTemplateRunAction(clientConfig),
				// },
			},
		},
	}
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "verbose, v",
			Usage: "show debug messages",
		},
		cli.IntFlag{
			Name:  "timeout",
			Usage: "Timeout (in seconds) for API request status polling.",
			Value: 60,
		},
		cli.StringFlag{
			Name:  "certPath, c",
			Usage: "HTTPS x509 cert path",
		},
		cli.StringFlag{
			Name:  "keyPath, k",
			Usage: "HTTPS x509 key path",
		},
		cli.StringFlag{
			Name:  "apiSession, as",
			Usage: "your active api session",
		},
		cli.BoolFlag{
			Name:  "insecure",
			Usage: "Bypass TLS certificate validation",
		},
		cli.StringFlag{
			Name:  "fiatUser",
			Usage: "Username for Fiat auth",
		},
		cli.StringFlag{
			Name:  "fiatPass",
			Usage: "Password for Fiat auth",
		},
	}
	app.Before = func(cc *cli.Context) error {
		if cc.GlobalBool("verbose") {
			logrus.SetLevel(logrus.DebugLevel)
		}
		return nil
	}
	return app
}

func validateFileExists(name, f string) {
	if _, err := os.Stat(f); os.IsNotExist(err) {
		logrus.WithFields(logrus.Fields{
			"name": name,
			"file": f,
		}).Error("file does not exist")
	}
}
