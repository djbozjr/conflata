package conflata

import (
	"errors"
	"testing"
)

func TestErrorGroupFieldsCopy(t *testing.T) {
	group := &ErrorGroup{}
	appendFieldError(&group, FieldError{
		FieldPath: "Database.URL",
		Attempts: []AttemptError{
			{Source: SourceEnv, Identifier: "DATABASE_URL", Err: errors.New("missing")},
		},
	})
	fields := group.Fields()
	if len(fields) != 1 {
		t.Fatalf("expected 1 field error, got %d", len(fields))
	}
	fields[0].FieldPath = "mutated"
	if group.Fields()[0].FieldPath != "Database.URL" {
		t.Fatal("expected Fields to return copy")
	}
	if !group.Has() {
		t.Fatal("expected Has to be true")
	}
}

func TestAttemptErrorString(t *testing.T) {
	err := AttemptError{
		Source:     SourceEnv,
		Identifier: "FOO",
		Err:        errors.New("boom"),
	}
	if err.Error() != "env (FOO): boom" {
		t.Fatalf("unexpected error string: %s", err.Error())
	}
}
