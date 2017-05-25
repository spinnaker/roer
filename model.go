package roer

import "github.com/spinnaker/roer/spinnaker"

type PipelineTemplate struct {
	Schema        string                   `json:"schema"`
	ID            string                   `json:"id"`
	Metadata      PipelineTemplateMetadata `json:"metadata"`
	Protect       bool                     `json:"protect"`
	Configuration PipelineTemplateConfig   `json:"configuration"`
	Variables     []interface{}            `json:"variables"`
	Stages        []PipelineTemplateStage  `json:"stages"`
}

type PipelineTemplateMetadata struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Owner       string   `json:"owner"`
	Scopes      []string `json:"scopes"`
}

type PipelineTemplateConfig struct {
	ConcurrentExecutions map[string]bool          `json:"concurrentExecutions"`
	Triggers             []map[string]interface{} `json:"triggers"`
	Parameters           []map[string]interface{} `json:"parameters"`
	Notifications        []map[string]interface{} `json:"notifications"`
}

type PipelineTemplateStage struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	DependsOn []string               `json:"dependsOn"`
	Name      string                 `json:"name"`
	Config    map[string]interface{} `json:"config"`
}

type PipelineTemplateModule struct {
	ID         string                   `json:"id"`
	Usage      string                   `json:"usage"`
	Variables  []map[string]interface{} `json:"variables"`
	When       []string                 `json:"when"`
	Definition map[string]interface{}   `json:"definition"`
}

type PipelineTemplatePartial struct {
	ID        string                   `json:"id"`
	Usage     string                   `json:"usage"`
	Variables []map[string]interface{} `json:"variables"`
	Stages    []PipelineTemplateStage  `json:"stages"`
}

type PipelineConfiguration struct {
	Schema        string                          `json:"schema"`
	ID            string                          `json:"id"`
	Pipeline      PipelineConfigurationDefinition `json:"pipeline"`
	Configuration PipelineConfig                  `json:"configuration"`
	Stages        []PipelineTemplateStage         `json:"stages"`
	Modules       []PipelineTemplateModule        `json:"modules"`
	Partials      []PipelineTemplatePartial       `json:"partials"`
}

func (c PipelineConfiguration) ToClient() spinnaker.PipelineConfig {
	// TODO rz - Should move this mapping around into orca itself
	parallel, ok := c.Configuration.ConcurrentExecutions["parallel"]
	if !ok {
		parallel = true
	}
	limitConcurrent, ok := c.Configuration.ConcurrentExecutions["limitConcurrent"]
	if !ok {
		limitConcurrent = true
	}
	keepWaitingPipelines, ok := c.Configuration.ConcurrentExecutions["keepWaitingPipelines"]
	if !ok {
		keepWaitingPipelines = false
	}
	return spinnaker.PipelineConfig{
		Type:                 "templatedPipeline",
		Name:                 c.Pipeline.Name,
		Application:          c.Pipeline.Application,
		Description:          c.Configuration.Description,
		Parallel:             parallel,
		LimitConcurrent:      limitConcurrent,
		KeepWaitingPipelines: keepWaitingPipelines,
		Config:               c,
		Locked: spinnaker.PipelineLock{
			AllowUnlockUI: false,
			UI:            true,
			Description:   "Manual edits are not allowed on templated pipelines",
		},
	}
}

type PipelineConfig struct {
	Inherit              []string        `json:"inherit"`
	ConcurrentExecutions map[string]bool `json:"concurrentExecutions"`
	Triggers             []interface{}   `json:"triggers"`
	Parameters           []interface{}   `json:"parameters"`
	Notifications        []interface{}   `json:"notifications"`
	Description          string          `json:"description"`
}

type PipelineConfigurationDefinition struct {
	Application string                 `json:"application"`
	Name        string                 `json:"name"`
	Template    TemplateSource         `json:"template"`
	Variables   map[string]interface{} `json:"variables"`
}

type TemplateSource struct {
	Source string `json:"source"`
}
