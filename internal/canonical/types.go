// Package canonical defines the source-of-truth domain types for harness-sync.
package canonical

type Config struct {
	Paths            Paths    `yaml:"paths,omitempty"`
	ActiveProfile    string   `yaml:"active_profile"`
	EnabledHarnesses []string `yaml:"enabled_harnesses"`
}

type Paths struct {
	Skills       string `yaml:"skills,omitempty"`
	Agents       string `yaml:"agents,omitempty"`
	Instructions string `yaml:"instructions,omitempty"`
}

type Profile struct {
	Name      string     `yaml:"name"`
	Gateway   Gateway    `yaml:"gateway"`
	Upstreams []Upstream `yaml:"upstreams,omitempty"`
	Models    []Model    `yaml:"models"`
}

type Gateway struct {
	URL          string `yaml:"url"`
	Token        string `yaml:"token"`
	DefaultModel string `yaml:"default_model"`
}

type Upstream struct {
	Name    string `yaml:"name"`
	APIKey  string `yaml:"api_key,omitempty"`
	BaseURL string `yaml:"base_url,omitempty"`
}

type Model struct {
	ID    string `yaml:"id"`
	Alias string `yaml:"alias,omitempty"`
}

type MCPRegistry struct {
	Servers []MCPServer `yaml:"servers"`
}

type MCPServer struct {
	Env       map[string]string `yaml:"env,omitempty"`
	Headers   map[string]string `yaml:"headers,omitempty"`
	Name      string            `yaml:"name"`
	Command   string            `yaml:"command,omitempty"`
	URL       string            `yaml:"url,omitempty"`
	Transport string            `yaml:"transport,omitempty"`
	Args      []string          `yaml:"args,omitempty"`
}

type Skill struct {
	Name        string
	Description string
	Body        string
	Path        string
}

type Agent struct {
	Name        string
	Description string
	Body        string
	Path        string
}

// Rule is a topic-scoped instruction fragment (e.g. ~/.claude/rules/go.md).
// Claude Code auto-loads a directory of these; harnesses without a native rules
// directory receive the bodies folded into their global instructions file.
type Rule struct {
	Name        string
	Description string
	Body        string
	Path        string
}

type Instructions struct {
	PerHarness map[string]string
	Global     string
}

type Bundle struct {
	Config       Config
	Instructions Instructions
	Root         string
	Profile      Profile
	Skills       []Skill
	Agents       []Agent
	Rules        []Rule
	MCP          MCPRegistry
}

func (p *Profile) LookupModel(alias string) (Model, bool) {
	for _, m := range p.Models {
		if m.Alias == alias || m.ID == alias {
			return m, true
		}
	}
	return Model{}, false
}
