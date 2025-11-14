package conflata

import (
	"fmt"
	"strings"
)

// ValueSource identifies where a configuration value was attempted to be read
// from (environment, provider, decoder, tag parsing, etc.).
type ValueSource string

const (
	SourceEnv      ValueSource = "env"
	SourceProvider ValueSource = "provider"
	SourceDecoder  ValueSource = "decoder"
	SourceTag      ValueSource = "tag"
)

// AttemptError captures metadata about a failed attempt (environment lookup,
// provider fetch, decode) that occurred while resolving a single field.
type AttemptError struct {
	Source     ValueSource
	Identifier string
	Err        error
}

// Error implements the error interface.
func (a AttemptError) Error() string {
	if a.Identifier == "" {
		return fmt.Sprintf("%s: %v", a.Source, a.Err)
	}
	return fmt.Sprintf("%s (%s): %v", a.Source, a.Identifier, a.Err)
}

// FieldError aggregates all failed attempts for a field. When a field cannot be
// satisfied it may record multiple AttemptErrors that callers can inspect to
// decide how to handle the failure.
type FieldError struct {
	FieldPath string
	Attempts  []AttemptError
}

// Error implements the error interface.
func (f FieldError) Error() string {
	var b strings.Builder
	_, _ = fmt.Fprintf(&b, "%s: ", f.FieldPath)
	errorsText := make([]string, len(f.Attempts))
	for i, att := range f.Attempts {
		errorsText[i] = att.Error()
	}
	b.WriteString(strings.Join(errorsText, "; "))
	return b.String()
}

// ErrorGroup groups field errors discovered during a loader run. The group can
// be inspected to understand which fields failed and why.
type ErrorGroup struct {
	fields []FieldError
}

// Error implements the error interface.
func (g *ErrorGroup) Error() string {
	if g == nil || len(g.fields) == 0 {
		return ""
	}
	var parts = make([]string, len(g.fields))
	for i, fieldErr := range g.fields {
		parts[i] = fieldErr.Error()
	}
	return "conflata: configuration errors: " + strings.Join(parts, "; ")
}

// Fields returns a copy of the underlying FieldError slice for inspection.
func (g *ErrorGroup) Fields() []FieldError {
	if g == nil {
		return nil
	}
	out := make([]FieldError, len(g.fields))
	copy(out, g.fields)
	return out
}

// Has reports whether the group contains any field errors.
func (g *ErrorGroup) Has() bool {
	return g != nil && len(g.fields) > 0
}

// appendFieldError adds a field error to the group, instantiating it if necessary.
func appendFieldError(g **ErrorGroup, field FieldError) {
	if len(field.Attempts) == 0 {
		return
	}
	group := *g
	if group == nil {
		group = &ErrorGroup{}
	}
	group.fields = append(group.fields, field)
	*g = group
}
