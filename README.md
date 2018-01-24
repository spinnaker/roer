# roer

A thin CLI for Spinnaker.

This project is aimed to provide a thin, limited client that's suitable for
CI environments where you may want to publish Pipeline Templates or update
pipeline configurations in Spinnaker. For a CLI to help configure and operate
use [halyard][halyard]: config & operating utilities are not in Roer's scope.

You can download the most recent version from the [Releases][releases] tab.

# Usage

Make sure your Spinnaker installation has pipeline-templates enabled:

`hal config features edit --pipeline-templates true`

Export `SPINNAKER_API` pointing to your Gate API.

```
NAME:
   roer - Spinnaker CLI

USAGE:
   main [global options] command [command options] [arguments...]

VERSION:
   dev

COMMANDS:
     pipeline           pipeline tasks
     pipeline-template  pipeline template tasks
     help, h            Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --verbose, -v               show debug messages
   --certPath value, -c value  HTTPS x509 cert path
   --keyPath value, -k value   HTTPS x509 key path
   --version                   print the version
```

# Commands

## pipeline-template

```
NAME:
   roer pipeline-template - pipeline template tasks

USAGE:
   roer pipeline-template  command [command options] [arguments...]

VERSION:
   dev

COMMANDS:
     publish  publish a pipeline template
     plan     validate a pipeline template and or plan a configuration
     convert  converts an existing, non-templated pipeline config into a scaffolded template
```

Publish template for use:

```json
$ SPINNAKER_API=https://localhost:7002 \
  go run cmd/roer/main.go pipeline-template publish examples/wait-template.yml
```

Plan a pipeline run using the template (invalid config example):

```json
$ SPINNAKER_API=https://localhost:7002 \
  go run cmd/roer/main.go pipeline-template plan examples/wait-config-invalid.yml
{
  "errors": [
    {
      "location": "configuration:stages.noConfigStanza",
      "message": "Stage configuration is unset",
      "severity": "FATAL"
    },
    {
      "location": "configuration:stages.noConfigStanza",
      "message": "A configuration-defined stage should have either dependsOn or an inject rule defined",
      "severity": "WARN"
    }
  ],
  "message": "Pipeline template is invalid",
  "status": "BAD_REQUEST"
}
```

Plan a pipeline run using the template (valid config example):

```json
$ SPINNAKER_API=https://localhost:7002 \
  go run cmd/roer/main.go pipeline-template plan examples/wait-config.yml
{
  "application": "spintest",
  "id": "unknown",
  "keepWaitingPipelines": false,
  "limitConcurrent": true,
  "name": "mpt",
  "notifications": [],
  "parameterConfig": [],
  "stages": [
    {
      "id": "947eb68b-1b03-4f33-b7c2-b3fa38eeef94",
      "name": "wait",
      "refId": "wait",
      "requisiteStageRefIds": [],
      "type": "wait",
      "waitTime": 5
    }
  ],
  "trigger": {
    "parameters": {},
    "type": "manual",
    "user": "anonymous"
  }
}
```

## pipeline

Create or update a managed pipeline within an applicaiton:

```json
$ SPINNAKER_API=https://localhost:7002 \
  go run cmd/roer/main.go pipeline save examples/wait-config.yml
```


# Development

Install [glide][glide], then install the dependencies:

`$ glide i`

To run:

`$ go run cmd/roer/main.go`

# Extending

You can extend the interface, as well as inject your own HTTP client by providing
your own `main.go`. This can be useful if you need to provide custom auth logic,
or if you want to add new commands, but not contribute them directly to the
project.

[releases]: https://github.com/spinnaker/roer/releases
[glide]: https://github.com/Masterminds/glide
[halyard]: https://github.com/spinnaker/halyard
