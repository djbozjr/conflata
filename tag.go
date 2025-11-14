package conflata

import (
	"fmt"
	"strings"
)

// fieldTag describes how to load and decode a single struct field.
type fieldTag struct {
	EnvKey      string
	ProviderKey string
	BackendName string
	Format      string
}

func parseFieldTag(raw string) (fieldTag, error) {
	if raw == "" {
		return fieldTag{}, nil
	}
	opts := strings.Fields(raw)
	tag := fieldTag{}
	for _, opt := range opts {
		key, value, ok := strings.Cut(opt, ":")
		if !ok {
			return fieldTag{}, fmt.Errorf("invalid conflata tag component %q", opt)
		}
		key = strings.ToLower(strings.TrimSpace(key))
		value = strings.TrimSpace(value)
		switch key {
		case "env":
			tag.EnvKey = value
		case "provider":
			tag.ProviderKey = value
		case "backend":
			tag.BackendName = value
		case "format":
			tag.Format = strings.ToLower(value)
		default:
			return fieldTag{}, fmt.Errorf("unknown conflata tag key %q", key)
		}
	}
	return tag, nil
}
