package spinnaker

import (
	"github.com/Sirupsen/logrus"
	"github.com/mitchellh/mapstructure"
)

// M allows certain responses to contain untyped data (most Spinnaker interfaces)
type M map[string]interface{}

type templatedPipelineRequest struct {
	Type   string      `json:"type"`
	Config interface{} `json:"config"`
	Plan   bool        `json:"plan"`
}

// TemplatedPipelineErrorResponse is returned when a pipeline template is invalid
type TemplatedPipelineErrorResponse struct {
	Errors  []TemplatedPipelineError `json:"errors"`
	Message string                   `json:"message"`
	Status  string                   `json:"status"`
}

// TemplatedPipelineError represents a single validation error
type TemplatedPipelineError struct {
	Location     string                   `json:"location"`
	Message      string                   `json:"message"`
	Suggestion   string                   `json:"suggestion"`
	Cause        string                   `json:"cause"`
	Severity     string                   `json:"severity"`
	Detail       map[string]string        `json:"detail"`
	NestedErrors []TemplatedPipelineError `json:"nestedErrors"`
}

// TaskRefResponse represents a task ID URL response following a submitted
// orchestration.
type TaskRefResponse struct {
	Ref string `json:"ref"`
}

// ExecutionResponse wraps the generic response format of an orchestration
// execution.
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

// ExtractRetrofitError will attempt to find a Retrofit exception and decode
// it into a RetrofitErrorResponse. This method will fatally error if the decode
// cannot be performed successfully.
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

// ExecutionStep partially represents a single Orca execution step.
type ExecutionStep struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	StartTime int    `json:"startTime"`
	EndTime   int    `json:"endTime"`
	Status    string `json:"status"`
}

// ExecutionVariable represents a variable key/value pair from an execution.
type ExecutionVariable struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}

type exceptionVariable struct {
	Details RetrofitErrorResponse `mapstructure:"details"`
}

// RetrofitErrorResponse represents a Retrofit error.
type RetrofitErrorResponse struct {
	Error        string   `mapstructure:"error"`
	Errors       []string `mapstructure:"errors"`
	Kind         string   `mapstructure:"kind"`
	ResponseBody string   `mapstructure:"responseBody"`
	Status       int      `mapstructure:"status"`
	URL          string   `mapstructure:"url"`
}

type PipelineConfig struct {
	ID                   string                   `json:"id,omitempty"`
	Type                 string                   `json:"type,omitempty"`
	Name                 string                   `json:"name"`
	Application          string                   `json:"application"`
	Description          string                   `json:"description,omitempty"`
	ExecutionEngine      string                   `json:"executionEngine,omitempty"`
	Parallel             bool                     `json:"parallel"`
	LimitConcurrent      bool                     `json:"limitConcurrent"`
	KeepWaitingPipelines bool                     `json:"keepWaitingPipelines"`
	Locked               PipelineLock             `json:"locked"`
	Stages               []map[string]interface{} `json:"stages,omitempty"`
	Triggers             []map[string]interface{} `json:"triggers,omitempty"`
	Parameters           []map[string]interface{} `json:"parameterConfig,omitempty"`
	Notifications        []map[string]interface{} `json:"notifications,omitempty"`
	LastModifiedBy       string                   `json:"lastModifiedBy"`
	Config               interface{}              `json:"config,omitempty"`
}

type PipelineLock struct {
	AllowUnlockUI bool   `json:"allowUnlockUi"`
	Description   string `json:"description"`
	UI            bool   `json:"ui"`
}
