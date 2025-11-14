package vault

import (
	"context"
	"errors"
	"testing"

	vaultapi "github.com/hashicorp/vault/api"
)

type stubKV struct {
	data map[string]*vaultapi.KVSecret
	err  error
}

func (s stubKV) Get(ctx context.Context, path string) (*vaultapi.KVSecret, error) {
	if s.err != nil {
		return nil, s.err
	}
	if secret, ok := s.data[path]; ok {
		return secret, nil
	}
	return nil, errors.New("not found")
}

func TestProviderDefaultField(t *testing.T) {
	secret := &vaultapi.KVSecret{Data: map[string]any{"value": "demo"}}
	provider, err := New(stubKV{data: map[string]*vaultapi.KVSecret{"secret/data/demo": secret}})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	got, err := provider.Fetch(context.Background(), "secret/data/demo")
	if err != nil {
		t.Fatalf("Fetch error: %v", err)
	}
	if got != "demo" {
		t.Fatalf("expected demo, got %s", got)
	}
}

func TestProviderExplicitField(t *testing.T) {
	secret := &vaultapi.KVSecret{Data: map[string]any{"password": "p4ss"}}
	provider, err := New(stubKV{data: map[string]*vaultapi.KVSecret{"secret/data/auth": secret}}, WithField("password"))
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	got, err := provider.Fetch(context.Background(), "secret/data/auth")
	if err != nil {
		t.Fatalf("Fetch error: %v", err)
	}
	if got != "p4ss" {
		t.Fatalf("expected password, got %s", got)
	}
}

func TestProviderJSONFallback(t *testing.T) {
	secret := &vaultapi.KVSecret{Data: map[string]any{"user": "admin", "password": "secret"}}
	provider, err := New(stubKV{data: map[string]*vaultapi.KVSecret{"secret/data/db": secret}})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	got, err := provider.Fetch(context.Background(), "secret/data/db")
	if err != nil {
		t.Fatalf("Fetch error: %v", err)
	}
	if got != `{"password":"secret","user":"admin"}` && got != `{"user":"admin","password":"secret"}` {
		t.Fatalf("expected JSON payload, got %s", got)
	}
}

func TestProviderMissingField(t *testing.T) {
	secret := &vaultapi.KVSecret{Data: map[string]any{"other": "value"}}
	provider, err := New(stubKV{data: map[string]*vaultapi.KVSecret{"secret/data/app": secret}}, WithField("password"))
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if _, err := provider.Fetch(context.Background(), "secret/data/app"); err == nil {
		t.Fatal("expected error for missing field")
	}
}

func TestNewRequiresKV(t *testing.T) {
	if _, err := New(nil); err == nil {
		t.Fatal("expected error when KV is nil")
	}
}
