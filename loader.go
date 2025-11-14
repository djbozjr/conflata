package conflata

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
)

// Provider fetches configuration values from an external system such as Vault,
// AWS Secrets Manager, or GCP Secret Manager. Custom providers can be
// registered with WithProvider.
type Provider interface {
	Fetch(ctx context.Context, key string) (string, error)
}

// EnvLookupFunc describes how to look up environment variables. Override with
// WithEnvLookup when running in custom environments.
type EnvLookupFunc func(string) (string, bool)

// Loader populates configuration structs from environment variables and
// external providers according to the struct tags.
type Loader struct {
	envLookup       EnvLookupFunc
	providers       map[string]Provider
	defaultProvider string
	defaultFormat   string
	decoders        map[string]DecodeFunc
	prefixFunc      func() string
	suffixFunc      func() string
}

// New constructs a Loader with optional functional options.
func New(opts ...Option) *Loader {
	l := &Loader{
		envLookup:       os.LookupEnv,
		providers:       make(map[string]Provider),
		defaultProvider: "aws",
		defaultFormat:   "json",
		decoders:        make(map[string]DecodeFunc),
	}
	for name, dec := range builtinDecoders {
		l.decoders[name] = dec
	}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

// Load populates the provided struct pointer with configuration data. When one
// or more fields fail to load, the returned error will be an *ErrorGroup that
// can be inspected for per-field failures. Other fatal errors (such as passing
// a non-struct pointer) are returned directly.
func (l *Loader) Load(ctx context.Context, target any) error {
	if target == nil {
		return errors.New("conflata: target cannot be nil")
	}
	value := reflect.ValueOf(target)
	if value.Kind() != reflect.Pointer || value.IsNil() {
		return errors.New("conflata: target must be a non-nil pointer")
	}
	elem := value.Elem()
	if elem.Kind() != reflect.Struct {
		return errors.New("conflata: target must point to a struct")
	}
	var group *ErrorGroup
	l.walkStruct(ctx, elem, "", &group)
	if group != nil && group.Has() {
		return group
	}
	return nil
}

func (l *Loader) walkStruct(ctx context.Context, current reflect.Value, prefix string, group **ErrorGroup) {
	t := current.Type()
	for i := 0; i < current.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}
		fieldValue := current.Field(i)
		fieldPath := field.Name
		if prefix != "" {
			fieldPath = prefix + "." + fieldPath
		}
		tagValue := field.Tag.Get("conflata")
		if tagValue == "" {
			continue
		}
		tag, err := parseFieldTag(tagValue)
		if err != nil {
			appendFieldError(group, FieldError{
				FieldPath: fieldPath,
				Attempts: []AttemptError{{
					Source: SourceTag,
					Err:    err,
				}},
			})
			continue
		}
		if tag.EnvKey == "" && tag.ProviderKey == "" && !tag.HasDefault {
			appendFieldError(group, FieldError{
				FieldPath: fieldPath,
				Attempts: []AttemptError{{
					Source: SourceTag,
					Err:    errors.New("tag must specify env or provider"),
				}},
			})
			continue
		}
		if assigned, err := l.populateField(ctx, fieldValue, fieldPath, tag); err != nil {
			appendFieldError(group, *err)
		} else if assigned {
			l.descend(ctx, fieldValue, fieldPath, group)
		}
	}
}

func (l *Loader) descend(ctx context.Context, fieldValue reflect.Value, fieldPath string, group **ErrorGroup) {
	switch fieldValue.Kind() {
	case reflect.Struct:
		l.walkStruct(ctx, fieldValue, fieldPath, group)
	case reflect.Pointer:
		elemType := fieldValue.Type().Elem()
		if elemType.Kind() == reflect.Struct {
			if fieldValue.IsNil() {
				fieldValue.Set(reflect.New(elemType))
			}
			l.walkStruct(ctx, fieldValue.Elem(), fieldPath, group)
		}
	}
}

func (l *Loader) populateField(ctx context.Context, fieldValue reflect.Value, fieldPath string, tag fieldTag) (bool, *FieldError) {
	collector := newAttemptCollector(fieldPath)
	assign := func(raw string) error {
		return l.assignValue(fieldValue, raw, tag.Format)
	}
	for _, src := range l.sourcesFor(tag) {
		if src == nil {
			continue
		}
		if collector.try(ctx, src, assign) {
			return true, nil
		}
	}
	if tag.HasDefault {
		if err := l.assignValue(fieldValue, tag.DefaultValue, tag.Format); err != nil {
			collector.fail(SourceTag, "default", fmt.Errorf("default decode: %w", err))
			return false, collector.result()
		}
		return true, nil
	}
	return false, collector.result()
}

func (l *Loader) assignValue(field reflect.Value, raw string, format string) error {
	targetType := field.Type()
	ptr := false
	if targetType.Kind() == reflect.Pointer {
		ptr = true
		targetType = targetType.Elem()
	}
	resolvedFormat := strings.ToLower(format)
	if resolvedFormat == "" && l.defaultFormat != "" && needsStructuredFormat(targetType) {
		resolvedFormat = l.defaultFormat
	}

	var (
		result any
		err    error
	)
	if resolvedFormat != "" {
		decoder, ok := l.decoders[resolvedFormat]
		if !ok {
			return fmt.Errorf("unknown format %q", resolvedFormat)
		}
		result, err = decoder(raw, targetType)
	} else {
		result, err = l.defaultDecode(raw, targetType)
	}
	if err != nil {
		return err
	}
	value := reflect.ValueOf(result)
	if !value.IsValid() {
		return errors.New("decoder produced invalid value")
	}
	if !value.Type().AssignableTo(targetType) {
		if value.Type().ConvertibleTo(targetType) {
			value = value.Convert(targetType)
		} else {
			return fmt.Errorf("decoder produced %s, cannot assign to %s", value.Type(), targetType)
		}
	}
	if ptr {
		if field.IsNil() {
			field.Set(reflect.New(targetType))
		}
		field.Elem().Set(value)
	} else {
		field.Set(value)
	}
	return nil
}

func (l *Loader) defaultDecode(raw string, targetType reflect.Type) (any, error) {
	switch targetType.Kind() {
	case reflect.Struct, reflect.Map, reflect.Slice, reflect.Array, reflect.Interface:
		return decodeJSON(raw, targetType)
	default:
		return decodePrimitive(raw, targetType)
	}
}

func needsStructuredFormat(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Struct, reflect.Map, reflect.Interface:
		return true
	case reflect.Slice:
		return t.Elem().Kind() != reflect.Uint8
	case reflect.Array:
		return true
	default:
		return false
	}
}
