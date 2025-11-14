package conflata

import "testing"

func TestParseFieldTagSuccess(t *testing.T) {
	tag, err := parseFieldTag(`env:DATABASE_URL provider:db-url backend:vault format:json`)
	if err != nil {
		t.Fatalf("parseFieldTag error: %v", err)
	}
	if tag.EnvKey != "DATABASE_URL" {
		t.Fatalf("expected env key DATABASE_URL, got %s", tag.EnvKey)
	}
	if tag.ProviderKey != "db-url" {
		t.Fatalf("expected provider key db-url, got %s", tag.ProviderKey)
	}
	if tag.BackendName != "vault" {
		t.Fatalf("expected backend vault, got %s", tag.BackendName)
	}
	if tag.Format != "json" {
		t.Fatalf("expected format json, got %s", tag.Format)
	}
}

func TestParseFieldTagUnknownKey(t *testing.T) {
	if _, err := parseFieldTag(`env:FOO foo:bar`); err == nil {
		t.Fatal("expected error for unknown key")
	}
}

func TestParseFieldTagMalformedComponent(t *testing.T) {
	if _, err := parseFieldTag(`envFOO`); err == nil {
		t.Fatal("expected error for malformed component")
	}
}
