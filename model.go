package tiller

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
