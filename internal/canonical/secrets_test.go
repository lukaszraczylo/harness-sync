package canonical

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// lookup returns env values from the provided map; missing keys return false.
func mapLookup(m map[string]string) func(string) (string, bool) {
	return func(name string) (string, bool) {
		v, ok := m[name]
		return v, ok
	}
}

func TestSubstituteSecretsGatewayToken(t *testing.T) {
	b := &Bundle{
		Profile: Profile{
			Name: "test",
			Gateway: Gateway{
				Token: "${MY_TOKEN}",
				URL:   "https://gw.lan",
			},
		},
	}
	err := SubstituteSecrets(b, mapLookup(map[string]string{"MY_TOKEN": "secret123"}))
	require.NoError(t, err)
	assert.Equal(t, "secret123", b.Profile.Gateway.Token)
	assert.Equal(t, "https://gw.lan", b.Profile.Gateway.URL)
}

func TestSubstituteSecretsUpstreamAPIKey(t *testing.T) {
	b := &Bundle{
		Profile: Profile{
			Name: "test",
			Upstreams: []Upstream{
				{Name: "openai", APIKey: "${OPENAI_KEY}", BaseURL: "https://api.openai.com"},
			},
		},
	}
	err := SubstituteSecrets(b, mapLookup(map[string]string{"OPENAI_KEY": "sk-abc"}))
	require.NoError(t, err)
	assert.Equal(t, "sk-abc", b.Profile.Upstreams[0].APIKey)
	assert.Equal(t, "https://api.openai.com", b.Profile.Upstreams[0].BaseURL)
}

func TestSubstituteSecretsMCPEnv(t *testing.T) {
	b := &Bundle{
		MCP: MCPRegistry{
			Servers: []MCPServer{
				{Name: "myserver", Env: map[string]string{"API_KEY": "${SRV_KEY}"}},
			},
		},
	}
	err := SubstituteSecrets(b, mapLookup(map[string]string{"SRV_KEY": "srv-secret"}))
	require.NoError(t, err)
	assert.Equal(t, "srv-secret", b.MCP.Servers[0].Env["API_KEY"])
}

func TestSubstituteSecretsMCPArgs(t *testing.T) {
	b := &Bundle{
		MCP: MCPRegistry{
			Servers: []MCPServer{
				{Name: "myserver", Args: []string{"--token", "${MCP_TOKEN}", "--plain"}},
			},
		},
	}
	err := SubstituteSecrets(b, mapLookup(map[string]string{"MCP_TOKEN": "tok"}))
	require.NoError(t, err)
	assert.Equal(t, []string{"--token", "tok", "--plain"}, b.MCP.Servers[0].Args)
}

func TestSubstituteSecretsMissingVarErrors(t *testing.T) {
	b := &Bundle{
		Profile: Profile{
			Name: "test",
			Gateway: Gateway{
				Token: "${MISSING_TOKEN}",
			},
		},
	}
	err := SubstituteSecrets(b, mapLookup(map[string]string{}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "MISSING_TOKEN")
	assert.Contains(t, err.Error(), "profile.gateway.token")
}

func TestSubstituteSecretsSkillBodyUntouched(t *testing.T) {
	b := &Bundle{
		Profile: Profile{Name: "test"},
		Skills: []Skill{
			{Name: "mys", Body: "Use ${FOO} in your prompt."},
		},
	}
	err := SubstituteSecrets(b, mapLookup(map[string]string{}))
	require.NoError(t, err)
	// skill body must be unchanged — ${FOO} is content, not config
	assert.Equal(t, "Use ${FOO} in your prompt.", b.Skills[0].Body)
}
