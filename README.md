# tiller

An unofficial, thin CLI for Spinnaker.

My objective in this project is to provide a client that's suitable for CI
environments where you may want to publish Pipeline Templates, etc into 
Spinnaker. For a CLI to help configure and operate Spinnaker itself, use 
[halyard][halyard]; config & operating utilities is not in Tiller's scope.

You can download the most recent version from the [Releases][releases] tab.

# Usage

Export `SPINNAKER_API` pointing to your Gate API.

```
NAME:
   tiller - Spinnaker CLI

USAGE:
   main [global options] command [command options] [arguments...]

VERSION:
   dev

COMMANDS:
     pipeline-template  pipeline template tasks
     help, h            Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --verbose, -v               show debug messages
   --silent, -s                silence log messages except panics
   --certPath value, -c value  HTTPS x509 cert path
   --keyPath value, -k value   HTTPS x509 key path
   --version                   print the version
```

# Commands

## pipeline-template

```
NAME:
   tiller pipeline-template - pipeline template tasks

USAGE:
   tiller pipeline-template  command [command options] [arguments...]

VERSION:
   dev

COMMANDS:
     publish  publish a pipeline template
     plan     validate a pipeline template and or plan a configuration
```

```json
$ SPINNAKER_API=https://localhost:7002 \
  go run cmd/tiller/main.go pipeline-template plan examples/wait-config-invalid.yml
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

# Development

Install [glide][glide], then install the dependencies:

`$ glide i`

# Extending

You can extend the interface, as well as inject your own HTTP client by providing
your own `main.go`. This can be useful if you need to provide custom auth logic,
or if you want to add new commands, but not contribute them directly to the
project.

[releases]: https://github.com/robzienert/tiller/releases
[glide]: https://github.com/Masterminds/glide
[halyard]: https://github.com/spinnaker/halyard
