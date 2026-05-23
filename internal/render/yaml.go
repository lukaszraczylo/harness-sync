package render

import "gopkg.in/yaml.v3"

func YAML(v any) ([]byte, error) {
	return yaml.Marshal(v)
}
