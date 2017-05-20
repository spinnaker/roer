package main

import (
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/robzienert/tiller/cmd"
	"github.com/robzienert/tiller/spinnaker"
)

// version is set via ldflags
var version = "dev"

func main() {
	// TODO rz - Don't really like this bit. Standardize a spinnaker config file.
	// maybe worthwhile splitting out this spinnaker API into a standard lib...
	if os.Getenv("SPINNAKER_API") == "" {
		logrus.Fatal("SPINNAKER_API must be set")
	}

	config := spinnaker.ClientConfig{
		Endpoint:          os.Getenv("SPINNAKER_API"),
		HTTPClientFactory: spinnaker.DefaultHTTPClientFactory,
	}
	if err := cmd.NewTiller(version, config).Run(os.Args); err != nil {
		os.Exit(1)
	}
}
