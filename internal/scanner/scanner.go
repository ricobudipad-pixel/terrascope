package scanner

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Resource struct {
	Type       string                 `json:"type"`
	Name       string                 `json:"name"`
	Properties map[string]interface{} `json:"properties"`
}

type DriftResult struct {
	Resource    string
	DriftType   string
	Severity    string
	Title       string
	Description string
	CurrentValue string
	ExpectedValue string
	Remediation string
}

func ParseTerraform(content string) []Resource {
	var resources []Resource
	lines := strings.Split(content, "\n")
	var current *Resource
	depth := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "resource ") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 3 {
				resType := strings.Trim(parts[1], "\"")
				resName := strings.Trim(parts[2], "\"")
				current = &Resource{
					Type:       resType,
					Name:       resName,
					Properties: make(map[string]interface{}),
				}
				depth = 0
			}
		}
		if current != nil {
			if strings.Contains(trimmed, "{") {
				depth++
			}
			if strings.Contains(trimmed, "}") {
				depth--
				if depth <= 0 {
					resources = append(resources, *current)
					current = nil
				}
			}
			// Simple key = value parsing
			if kv := strings.SplitN(trimmed, "=", 2); len(kv) == 2 && current != nil {
				key := strings.TrimSpace(kv[0])
				val := strings.TrimSpace(strings.Trim(kv[1], "\""))
				if key != "" && !strings.Contains(key, "{") && !strings.Contains(key, "}") {
					current.Properties[key] = val
				}
			}
		}
	}
	return resources
}

func ParseKubernetes(content string) []Resource {
	var resources []Resource
	// Simple YAML parsing for K8s resources
	docs := strings.Split(content, "---")
	for _, doc := range docs {
		doc = strings.TrimSpace(doc)
		if doc == "" {
			continue
		}
		res := Resource{Properties: make(map[string]interface{})}
		lines := strings.Split(doc, "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "kind:") {
				res.Type = strings.TrimSpace(strings.TrimPrefix(trimmed, "kind:"))
			}
			if strings.HasPrefix(trimmed, "name:") && res.Name == "" {
				res.Name = strings.TrimSpace(strings.TrimPrefix(trimmed, "name:"))
			}
		}
		if res.Type != "" {
			resources = append(resources, res)
		}
	}
	return resources
}

func DetectDrift(baseline, current []Resource) []DriftResult {
	var drifts []DriftResult
	baseMap := make(map[string]Resource)
	for _, r := range baseline {
		key := fmt.Sprintf("%s.%s", r.Type, r.Name)
		baseMap[key] = r
	}

	for _, r := range current {
		key := fmt.Sprintf("%s.%s", r.Type, r.Name)
		if base, ok := baseMap[key]; ok {
			// Check for property drift
			for k, v := range r.Properties {
				if bv, exists := base.Properties[k]; exists {
					if fmt.Sprintf("%v", v) != fmt.Sprintf("%v", bv) {
						drifts = append(drifts, DriftResult{
							Resource:    key,
							DriftType:   "modified",
							Severity:    classifySeverity(k),
							Title:       fmt.Sprintf("%s modified: %s", key, k),
							Description: fmt.Sprintf("Property %s changed from %v to %v", k, bv, v),
							CurrentValue: fmt.Sprintf("%v", v),
							ExpectedValue: fmt.Sprintf("%v", bv),
							Remediation: fmt.Sprintf("Revert %s to %v or update baseline", k, bv),
						})
					}
				}
			}
			delete(baseMap, key)
		} else {
			drifts = append(drifts, DriftResult{
				Resource:  key,
				DriftType: "added",
				Severity:  "medium",
				Title:     fmt.Sprintf("New resource: %s", key),
				Description: "Resource not found in baseline",
			})
		}
	}

	// Remaining baseline resources are missing from current
	for key := range baseMap {
		drifts = append(drifts, DriftResult{
			Resource:  key,
			DriftType: "missing",
			Severity:  "high",
			Title:     fmt.Sprintf("Missing resource: %s", key),
			Description: "Resource exists in baseline but not in current config",
			Remediation: "Restore the resource or update baseline",
		})
	}
	return drifts
}

func classifySeverity(property string) string {
	lower := strings.ToLower(property)
	if strings.Contains(lower, "cidr") || strings.Contains(lower, "public") || strings.Contains(lower, "0.0.0.0") {
		return "critical"
	}
	if strings.Contains(lower, "security") || strings.Contains(lower, "acl") || strings.Contains(lower, "encryption") {
		return "high"
	}
	if strings.Contains(lower, "instance_type") || strings.Contains(lower, "replicas") {
		return "medium"
	}
	return "low"
}

func ToJSON(resources []Resource) string {
	b, _ := json.MarshalIndent(resources, "", "  ")
	return string(b)
}
