package spinnaker

import (
	"github.com/Sirupsen/logrus"
	"github.com/mitchellh/mapstructure"
)

type templatedPipelineRequest struct {
	Type   string      `json:"type"`
	Config interface{} `json:"config"`
	Plan   bool        `json:"plan"`
}

type TemplatedPipelineErrorResponse struct {
	Errors  []TemplatedPipelineError `json:"errors"`
	Message string                   `json:"message"`
	Status  string                   `json:"status"`
}

type TemplatedPipelineError struct {
	Location     string                   `json:"location"`
	Message      string                   `json:"message"`
	Suggestion   string                   `json:"suggestion"`
	Cause        string                   `json:"cause"`
	Severity     string                   `json:"severity"`
	Detail       map[string]string        `json:"detail"`
	NestedErrors []TemplatedPipelineError `json:"nestedErrors"`
}

type TaskRefResponse struct {
	Ref string `json:"ref"`
}

type ExecutionResponse struct {
	ID          string              `json:"id"`
	Name        string              `json:"string"`
	Application string              `json:"application"`
	Status      string              `json:"status"`
	BuildTime   int                 `json:"buildTime"`
	StartTime   int                 `json:"startTime"`
	EndTime     int                 `json:"endTime"`
	Execution   interface{}         `json:"execution"`
	Steps       []ExecutionStep     `json:"steps"`
	Variables   []ExecutionVariable `json:"variables"`
}

func (e ExecutionResponse) ExtractRetrofitError() *RetrofitErrorResponse {
	for _, v := range e.Variables {
		if v.Key == "exception" {
			var exception exceptionVariable
			if err := mapstructure.Decode(v.Value, &exception); err != nil {
				logrus.WithError(err).Fatal("could not decode exception struct")
			}
			return &exception.Details
		}
	}
	return nil
}

type ExecutionStep struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	StartTime int    `json:"startTime"`
	EndTime   int    `json:"endTime"`
	Status    string `json:"status"`
}

type ExecutionVariable struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}

type exceptionVariable struct {
	Details RetrofitErrorResponse `mapstructure:"details"`
}

type RetrofitErrorResponse struct {
	Error        string   `mapstructure:"error"`
	Errors       []string `mapstructure:"errors"`
	Kind         string   `mapstructure:"kind"`
	ResponseBody string   `mapstructure:"responseBody"`
	Status       int      `mapstructure:"status"`
	URL          string   `mapstructure:"url"`
}
