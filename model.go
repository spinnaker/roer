package roer

import "github.com/spinnaker/roer/spinnaker"

// PipelineTemplate is a pipeline template
type PipelineTemplate struct {
	Schema        string                   `json:"schema"`
	ID            string                   `json:"id"`
	Metadata      PipelineTemplateMetadata `json:"metadata"`
	Protect       bool                     `json:"protect"`
	Configuration PipelineTemplateConfig   `json:"configuration,omitempty"`
	Variables     []interface{}            `json:"variables,omitempty"`
	Stages        []PipelineTemplateStage  `json:"stages"`
}

// PipelineTemplateMetadata metadata for a template
type PipelineTemplateMetadata struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Owner       string   `json:"owner"`
	Scopes      []string `json:"scopes,omitempty"`
}

// PipelineTemplateConfig pipeline template config
type PipelineTemplateConfig struct {
	ConcurrentExecutions map[string]bool          `json:"concurrentExecutions,omitempty"`
	Triggers             []map[string]interface{} `json:"triggers,omitempty"`
	ExpectedArtifacts    []map[string]interface{} `json:"expectedArtifacts,omitempty"`
	Parameters           []map[string]interface{} `json:"parameters,omitempty"`
	Notifications        []map[string]interface{} `json:"notifications,omitempty"`
}

// PipelineTemplateStage a pipeline template stage
type PipelineTemplateStage struct {
	ID                 string                                  `json:"id"`
	Type               string                                  `json:"type"`
	DependsOn          []string                                `json:"dependsOn,omitempty"`
	Inject             PipelineTemplateStageInjection          `json:"inject,omitempty"`
	Name               string                                  `json:"name"`
	Config             map[string]interface{}                  `json:"config"`
	Notifications      []map[string]interface{}                `json:"notifications,omitempty"`
	Comments           string                                  `json:"comments,omitempty"`
	When               []string                                `json:"when,omitempty"`
	InheritanceControl PipelineTemplateStageInheritanceControl `json:"inheritanceControl,omitempty"`
}

// PipelineTemplateStageInjection is a pipeline template stage injection
type PipelineTemplateStageInjection struct {
	First  bool     `json:"first,omitempty"`
	Last   bool     `json:"last,omitempty"`
	Before []string `json:"before,omitempty"`
	After  []string `json:"after,omitempty"`
}

// PipelineTemplateStageInheritanceControl pipeline template stage inheritance control
type PipelineTemplateStageInheritanceControl struct {
	Merge   []InheritanceControlRule `json:"merge,omitempty"`
	Replace []InheritanceControlRule `json:"replace,omitempty"`
	Remove  []InheritanceControlRule `json:"remove,omitempty"`
}

// InheritanceControlRule inheritance control rule
type InheritanceControlRule struct {
	Path  string      `json:"path"`
	Value interface{} `json:"value"`
}

// PipelineTemplateModule pipeline template module
type PipelineTemplateModule struct {
	ID         string                   `json:"id"`
	Usage      string                   `json:"usage"`
	Variables  []map[string]interface{} `json:"variables,omitempty"`
	When       []string                 `json:"when,omitempty"`
	Definition map[string]interface{}   `json:"definition"`
}

// PipelineTemplatePartial pipeline template partial
type PipelineTemplatePartial struct {
	ID        string                   `json:"id"`
	Usage     string                   `json:"usage"`
	Variables []map[string]interface{} `json:"variables,omitempty"`
	Stages    []PipelineTemplateStage  `json:"stages"`
}

// PipelineConfiguration pipeline configuration
type PipelineConfiguration struct {
	Schema        string                          `json:"schema"`
	ID            string                          `json:"id"`
	Pipeline      PipelineConfigurationDefinition `json:"pipeline"`
	Configuration PipelineConfig                  `json:"configuration"`
	Stages        []PipelineTemplateStage         `json:"stages"`
	Modules       []PipelineTemplateModule        `json:"modules,omitempty"`
	Partials      []PipelineTemplatePartial       `json:"partials,omitempty"`
}

// ToClient convert PipelineConfiguration to spinnaker.PipelineConfig
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
		ID:                   c.Pipeline.PipelineConfigID,
		Type:                 "templatedPipeline",
		Name:                 c.Pipeline.Name,
		Application:          c.Pipeline.Application,
		Description:          c.Configuration.Description,
		Parallel:             parallel,
		LimitConcurrent:      limitConcurrent,
		KeepWaitingPipelines: keepWaitingPipelines,
		Config:               c,
	}
}

// PipelineConfig pipeline config
type PipelineConfig struct {
	Inherit              []string        `json:"inherit"`
	ConcurrentExecutions map[string]bool `json:"concurrentExecutions"`
	Triggers             []interface{}   `json:"triggers"`
	ExpectedArtifacts    []interface{}   `json:"expectedArtifacts"`
	Parameters           []interface{}   `json:"parameters"`
	Notifications        []interface{}   `json:"notifications"`
	Description          string          `json:"description"`
}

// PipelineConfigurationDefinition pipline config definition
type PipelineConfigurationDefinition struct {
	Application      string                 `json:"application"`
	Name             string                 `json:"name"`
	Template         TemplateSource         `json:"template"`
	PipelineConfigID string                 `json:"pipelineConfigId"`
	Variables        map[string]interface{} `json:"variables"`
}

// TemplateSource template source
type TemplateSource struct {
	Source string `json:"source"`
}
