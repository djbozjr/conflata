package conflata

import (
	"context"
	"errors"
)

type valueSource interface {
	Source() ValueSource
	Identifier() string
	Fetch(ctx context.Context) (string, error)
}

type attemptCollector struct {
	fieldPath string
	attempts  []AttemptError
}

func newAttemptCollector(fieldPath string) *attemptCollector {
	return &attemptCollector{fieldPath: fieldPath}
}

func (c *attemptCollector) try(ctx context.Context, src valueSource, assign func(string) error) bool {
	raw, err := src.Fetch(ctx)
	if err != nil {
		c.fail(src.Source(), src.Identifier(), err)
		return false
	}
	if err := assign(raw); err != nil {
		c.fail(SourceDecoder, src.Identifier(), err)
		return false
	}
	return true
}

func (c *attemptCollector) fail(source ValueSource, identifier string, err error) {
	c.attempts = append(c.attempts, AttemptError{
		Source:     source,
		Identifier: identifier,
		Err:        err,
	})
}

func (c *attemptCollector) result() *FieldError {
	if len(c.attempts) == 0 {
		c.fail(SourceTag, "", errors.New("no env or provider attempts recorded"))
	}
	return &FieldError{
		FieldPath: c.fieldPath,
		Attempts:  c.attempts,
	}
}
