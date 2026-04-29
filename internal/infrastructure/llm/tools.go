package llm

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/enterprise-ai/orchestrator/internal/domain/models"
)

type providerTool struct {
	Type     string           `json:"type"`
	Function providerFunction `json:"function"`
}

type providerFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters"`
}

func toProviderTools(tools []models.Tool) []providerTool {
	if len(tools) == 0 {
		return nil
	}

	out := make([]providerTool, 0, len(tools))
	for _, tool := range tools {
		tool.Name = strings.TrimSpace(tool.Name)
		if tool.Name == "" {
			continue
		}

		properties := make(map[string]any, len(tool.Parameters))
		var required []string
		for name, param := range tool.Parameters {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			paramType := strings.TrimSpace(param.Type)
			if paramType == "" {
				paramType = "string"
			}
			properties[name] = map[string]any{
				"type":        paramType,
				"description": param.Description,
			}
			if param.Required {
				required = append(required, name)
			}
		}

		parameters := map[string]any{
			"type":                 "object",
			"properties":           properties,
			"additionalProperties": false,
		}
		if len(required) > 0 {
			parameters["required"] = required
		}

		out = append(out, providerTool{
			Type: "function",
			Function: providerFunction{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  parameters,
			},
		})
	}
	return out
}

func parseToolArguments(raw any) (map[string]any, error) {
	switch v := raw.(type) {
	case nil:
		return map[string]any{}, nil
	case map[string]any:
		return v, nil
	case string:
		if strings.TrimSpace(v) == "" {
			return map[string]any{}, nil
		}
		var args map[string]any
		if err := json.Unmarshal([]byte(v), &args); err != nil {
			return nil, fmt.Errorf("parse tool arguments: %w", err)
		}
		return args, nil
	case json.RawMessage:
		if len(v) == 0 || string(v) == "null" {
			return map[string]any{}, nil
		}
		var args map[string]any
		if err := json.Unmarshal(v, &args); err != nil {
			return nil, fmt.Errorf("parse tool arguments: %w", err)
		}
		return args, nil
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("marshal tool arguments: %w", err)
		}
		var args map[string]any
		if err := json.Unmarshal(b, &args); err != nil {
			return nil, fmt.Errorf("parse tool arguments: %w", err)
		}
		return args, nil
	}
}
