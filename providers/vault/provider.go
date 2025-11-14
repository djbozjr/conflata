package vault

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	vaultapi "github.com/hashicorp/vault/api"
)

// KV is the subset of the Vault KV v2 interface the provider depends on.
type KV interface {
	Get(ctx context.Context, path string) (*vaultapi.KVSecret, error)
}

// Provider loads secrets from a Vault KV v2 mount.
type Provider struct {
	kv       KV
	field    string
	explicit bool
}

// Option configures the Vault provider.
type Option func(*Provider)

// WithField selects a concrete key in the Vault data map. When omitted,
// Conflata attempts helpful defaults such as "value" or serializing the entire
// map as JSON.
func WithField(field string) Option {
	return func(p *Provider) {
		p.field = field
		p.explicit = true
	}
}

// New creates a Vault provider using the given KV accessor.
func New(kv KV, opts ...Option) (*Provider, error) {
	if kv == nil {
		return nil, errors.New("vault: KV accessor is required")
	}
	p := &Provider{kv: kv}
	for _, opt := range opts {
		opt(p)
	}
	return p, nil
}

// FromClient is a convenience helper that derives a KV accessor from a Vault
// client and mount path.
func FromClient(client *vaultapi.Client, mountPath string, opts ...Option) (*Provider, error) {
	if client == nil {
		return nil, errors.New("vault: client is required")
	}
	if mountPath == "" {
		mountPath = "secret"
	}
	return New(client.KVv2(mountPath), opts...)
}

// Fetch retrieves the secret at the supplied path.
func (p *Provider) Fetch(ctx context.Context, path string) (string, error) {
	if path == "" {
		return "", errors.New("vault: secret path cannot be empty")
	}
	secret, err := p.kv.Get(ctx, path)
	if err != nil {
		return "", fmt.Errorf("vault: %w", err)
	}
	if secret == nil || secret.Data == nil {
		return "", errors.New("vault: secret contained no data")
	}
	return p.extract(secret.Data)
}

func (p *Provider) extract(data map[string]any) (string, error) {
	if len(data) == 0 {
		return "", errors.New("vault: secret data empty")
	}
	if p.explicit {
		value, ok := data[p.field]
		if !ok {
			return "", fmt.Errorf("vault: field %q not found", p.field)
		}
		return asString(value, p.field)
	}
	if value, ok := data["value"]; ok {
		if str, err := asString(value, "value"); err == nil {
			return str, nil
		}
	}
	if len(data) == 1 {
		for key, value := range data {
			if str, err := asString(value, key); err == nil {
				return str, nil
			}
		}
	}
	buf, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("vault: marshal secret: %w", err)
	}
	return string(buf), nil
}

func asString(value any, field string) (string, error) {
	switch v := value.(type) {
	case string:
		return v, nil
	case fmt.Stringer:
		return v.String(), nil
	default:
		return "", fmt.Errorf("vault: field %q is not a string", field)
	}
}
