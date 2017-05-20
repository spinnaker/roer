package cmd

import (
	"errors"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/robzienert/tiller"
	"github.com/robzienert/tiller/spinnaker"
	"github.com/urfave/cli"
)

// NewTiller returns a new instance of the OSS Tiller application
func NewTiller(version string, clientConfig spinnaker.ClientConfig) *cli.App {
	cli.VersionFlag.Name = "version"
	cli.HelpFlag.Name = "help"
	cli.HelpFlag.Hidden = true

	app := cli.NewApp()
	app.Name = "tiller"
	app.Usage = "Spinnaker CLI"
	app.Version = version
	app.Commands = []cli.Command{
		{
			Name:  "pipeline-template",
			Usage: "pipeline template tasks",
			Subcommands: []cli.Command{
				{
					Name:      "publish",
					Usage:     "publish a pipeline template",
					ArgsUsage: "[template.yml]",
					Before: func(cc *cli.Context) error {
						if cc.NArg() != 1 {
							return errors.New("path to template file is required")
						}
						return nil
					},
					Flags: []cli.Flag{
						cli.BoolFlag{
							Name:  "update, u",
							Usage: "update the given pipeline",
						},
					},
					Action: tiller.PipelineTemplatePublishAction(clientConfig),
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
					Before: func(cc *cli.Context) error {
						if cc.NArg() != 1 {
							return errors.New("path to configuration file is required")
						}
						return nil
					},
					Action: tiller.PipelineTemplatePlanAction(clientConfig),
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
				// 	Action: tiller.PipelineTemplateRunAction(clientConfig),
				// },
			},
		},
	}
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "verbose, v",
			Usage: "show debug messages",
		},
		cli.BoolFlag{
			Name:  "silent, s",
			Usage: "silence log messages except panics",
		},
		cli.StringFlag{
			Name:  "certPath, c",
			Usage: "HTTPS x509 cert path",
		},
		cli.StringFlag{
			Name:  "keyPath, k",
			Usage: "HTTPS x509 key path",
		},
	}
	app.Before = func(cc *cli.Context) error {
		if cc.Bool("verbose") {
			logrus.SetLevel(logrus.DebugLevel)
		}
		if hasX509Files(cc, true) {
			logrus.WithFields(logrus.Fields{
				"certPath": cc.String("certPath"),
				"keyPath":  cc.String("keyPath"),
			}).Error("certPath and keyPath must be defined")
		}
		if hasX509Files(cc, false) {
			validateFileExists("certPath", cc.String("certPath"))
			validateFileExists("keyPath", cc.String("keyPath"))
		}
		return nil
	}
	return app
}

func hasX509Files(cc *cli.Context, any bool) bool {
	if any {
		return cc.IsSet("certPath") || cc.IsSet("keyPath")
	}
	return cc.IsSet("certPath") && cc.IsSet("keyPath")
}

func validateFileExists(name, f string) {
	if _, err := os.Stat(f); os.IsNotExist(err) {
		logrus.WithFields(logrus.Fields{
			"name": name,
			"file": f,
		}).Error("file does not exist")
	}
}
