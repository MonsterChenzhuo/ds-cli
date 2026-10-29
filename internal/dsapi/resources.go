package dsapi

import (
	"fmt"
	"strings"
)

// ResourceReference uses the storage path returned by DS resources APIs.
// The worker downloads this path preserving the resource's relative directory.
type ResourceReference struct {
	ResourceName string `json:"resourceName"`
}

func ValidateResourceReferences(resources []ResourceReference) error {
	seen := make(map[string]bool, len(resources))
	for i, resource := range resources {
		name := resource.ResourceName
		if strings.TrimSpace(name) == "" || name != strings.TrimSpace(name) {
			return fmt.Errorf("resource[%d].resourceName must be a non-empty storage path without surrounding whitespace", i)
		}
		if seen[name] {
			return fmt.Errorf("duplicate resourceName: %s", name)
		}
		seen[name] = true
	}
	return nil
}
