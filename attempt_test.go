package conflata

import (
	"context"
	"errors"
	"testing"
)

type stubSource struct {
	value      string
	err        error
	identifier string
	source     ValueSource
}

func (s stubSource) Source() ValueSource { return s.source }
func (s stubSource) Identifier() string  { return s.identifier }
func (s stubSource) Fetch(context.Context) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	return s.value, nil
}

func TestAttemptCollectorTryStopsOnSuccess(t *testing.T) {
	c := newAttemptCollector("Field")
	assignments := 0
	assign := func(string) error {
		assignments++
		return nil
	}

	srcs := []valueSource{
		stubSource{err: errors.New("boom"), identifier: "env", source: SourceEnv},
		stubSource{value: "value", identifier: "provider:secret", source: SourceProvider},
	}

	var succeeded bool
	for _, src := range srcs {
		if c.try(context.Background(), src, assign) {
			succeeded = true
			break
		}
	}
	if !succeeded {
		t.Fatal("expected success from second source")
	}
	if assignments != 1 {
		t.Fatalf("expected one assignment, got %d", assignments)
	}
}

func TestAttemptCollectorResultAddsTagErrorWhenEmpty(t *testing.T) {
	c := newAttemptCollector("Field")
	err := c.result()
	if len(err.Attempts) != 1 || err.Attempts[0].Source != SourceTag {
		t.Fatalf("expected SourceTag fallback, got %+v", err.Attempts)
	}
}
