package conflata

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// fieldTag describes how to load and decode a single struct field.
type fieldTag struct {
	EnvKey       string
	ProviderKey  string
	BackendName  string
	Format       string
	DefaultValue string
	HasDefault   bool
}

func parseFieldTag(raw string) (fieldTag, error) {
	if raw == "" {
		return fieldTag{}, nil
	}
	var (
		tag        fieldTag
		keyBuilder strings.Builder
		valBuilder strings.Builder
		currentKey string
		state      = stateKey
		quote      rune
		escape     bool
	)

	commit := func() error {
		if currentKey == "" {
			return fmt.Errorf("conflata: missing key before value %q", valBuilder.String())
		}
		value := valBuilder.String()
		valBuilder.Reset()
		if err := tag.assign(currentKey, value); err != nil {
			return err
		}
		currentKey = ""
		state = stateKey
		return nil
	}

	for i := 0; i < len(raw); {
		r, size := utf8.DecodeRuneInString(raw[i:])
		i += size

		switch state {
		case stateKey:
			if unicode.IsSpace(r) {
				continue
			}
			if r == ':' {
				currentKey = strings.ToLower(strings.TrimSpace(keyBuilder.String()))
				if currentKey == "" {
					return fieldTag{}, fmt.Errorf("conflata: empty tag key")
				}
				keyBuilder.Reset()
				state = statePreValue
				continue
			}
			keyBuilder.WriteRune(r)

		case statePreValue:
			if unicode.IsSpace(r) {
				continue
			}
			if r == '"' || r == '\'' {
				quote = r
				state = stateValueQuoted
				continue
			}
			valBuilder.WriteRune(r)
			state = stateValue

		case stateValue:
			if unicode.IsSpace(r) {
				if err := commit(); err != nil {
					return fieldTag{}, err
				}
				continue
			}
			valBuilder.WriteRune(r)

		case stateValueQuoted:
			if escape {
				valBuilder.WriteRune(r)
				escape = false
				continue
			}
			if r == '\\' {
				escape = true
				continue
			}
			if r == quote {
				quote = 0
				if err := commit(); err != nil {
					return fieldTag{}, err
				}
				continue
			}
			valBuilder.WriteRune(r)
		}
	}

	switch state {
	case stateKey:
		if keyBuilder.Len() != 0 {
			return fieldTag{}, fmt.Errorf("conflata: dangling key %q", keyBuilder.String())
		}
	case statePreValue:
		return fieldTag{}, fmt.Errorf("conflata: key %q missing value", currentKey)
	case stateValue:
		if err := commit(); err != nil {
			return fieldTag{}, err
		}
	case stateValueQuoted:
		return fieldTag{}, fmt.Errorf("conflata: unterminated quoted value for key %q", currentKey)
	}

	return tag, nil
}

func (t *fieldTag) assign(key, value string) error {
	switch key {
	case "env":
		t.EnvKey = value
	case "provider":
		t.ProviderKey = value
	case "backend":
		t.BackendName = value
	case "format":
		t.Format = strings.ToLower(value)
	case "default":
		t.DefaultValue = value
		t.HasDefault = true
	default:
		return fmt.Errorf("unknown conflata tag key %q", key)
	}
	return nil
}

const (
	stateKey = iota
	statePreValue
	stateValue
	stateValueQuoted
)
