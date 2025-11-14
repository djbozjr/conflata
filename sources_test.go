package conflata

import (
	"context"
	"errors"
	"testing"
)

type fakeProvider struct {
	value string
	err   error
}

func (f fakeProvider) Fetch(ctx context.Context, key string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.value, nil
}

func TestSourcesForEnvAndProvider(t *testing.T) {
	loader := New(
		WithEnvLookup(func(string) (string, bool) { return "env-value", true }),
	)
	loader.providers["vault"] = fakeProvider{value: "secret"}
	tag := fieldTag{
		EnvKey:      "FOO",
		ProviderKey: "bar",
		BackendName: "vault",
	}
	sources := loader.sourcesFor(tag)
	if len(sources) != 2 {
		t.Fatalf("expected 2 sources, got %d", len(sources))
	}
	if sources[0].Source() != SourceEnv {
		t.Fatalf("expected env first, got %s", sources[0].Source())
	}
	if sources[1].Source() != SourceProvider {
		t.Fatalf("expected provider second, got %s", sources[1].Source())
	}
}

func TestProviderSourceHandlesMissingProvider(t *testing.T) {
	loader := New()
	tag := fieldTag{ProviderKey: "secret", BackendName: "missing"}
	src := loader.newProviderSource(tag)
	if _, err := src.Fetch(context.Background()); err == nil {
		t.Fatal("expected error when provider missing")
	}
}

func TestProviderSourceEmptySecret(t *testing.T) {
	loader := New()
	loader.providers["vault"] = fakeProvider{value: ""}
	tag := fieldTag{ProviderKey: "secret", BackendName: "vault"}
	src := loader.newProviderSource(tag)
	if _, err := src.Fetch(context.Background()); err == nil {
		t.Fatal("expected error for empty secret payload")
	}
}

func TestProviderSourcePropagatesError(t *testing.T) {
	loader := New()
	loader.providers["vault"] = fakeProvider{err: errors.New("boom")}
	tag := fieldTag{ProviderKey: "secret", BackendName: "vault"}
	src := loader.newProviderSource(tag)
	if _, err := src.Fetch(context.Background()); err == nil {
		t.Fatal("expected provider error to surface")
	}
}
