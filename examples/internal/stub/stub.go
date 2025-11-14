package stub

import (
	"context"
	"errors"
	"os"

	"github.com/djbozjr/conflata"
)

// EnvKey toggles stub providers for examples.
const EnvKey = "CONFLATA_EXAMPLE_STUB"

// Enabled reports whether stub providers should be used.
func Enabled() bool {
	_, ok := os.LookupEnv(EnvKey)
	return ok
}

// PopulateEnv seeds environment variables to keep the examples runnable when
// stub providers are active.
func PopulateEnv(values map[string]string) {
	for key, value := range values {
		_ = os.Setenv(key, value)
	}
}

// NewProvider constructs a simple conflata.Provider backed by an in-memory map.
func NewProvider(values map[string]string) conflata.Provider {
	copied := make(map[string]string, len(values))
	for k, v := range values {
		copied[k] = v
	}
	return provider{secrets: copied}
}

type provider struct {
	secrets map[string]string
}

func (p provider) Fetch(ctx context.Context, key string) (string, error) {
	if value, ok := p.secrets[key]; ok {
		return value, nil
	}
	return "", errors.New("stub secret not found: " + key)
}
