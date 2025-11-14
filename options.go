package conflata

import "strings"

// Option configures the Loader.
type Option func(*Loader)

// WithProvider registers a provider under the supplied name so struct tags can
// reference it via the `backend:` key.
func WithProvider(name string, provider Provider) Option {
	return func(l *Loader) {
		if name == "" || provider == nil {
			return
		}
		if l.providers == nil {
			l.providers = make(map[string]Provider)
		}
		l.providers[strings.ToLower(name)] = provider
	}
}

// WithDefaultProvider picks which registered provider should be used when a tag
// does not specify a backend explicitly.
func WithDefaultProvider(name string) Option {
	return func(l *Loader) {
		l.defaultProvider = strings.ToLower(name)
	}
}

// WithEnvLookup overrides the environment variable lookup strategy.
func WithEnvLookup(fn EnvLookupFunc) Option {
	return func(l *Loader) {
		if fn != nil {
			l.envLookup = fn
		}
	}
}

// WithDecoder registers a custom format decoder keyed by name. Struct tags can
// then reference the decoder via `format:decoder`.
func WithDecoder(name string, fn DecodeFunc) Option {
	return func(l *Loader) {
		if name == "" || fn == nil {
			return
		}
		if l.decoders == nil {
			l.decoders = make(map[string]DecodeFunc)
		}
		l.decoders[strings.ToLower(name)] = fn
	}
}

// WithDefaultFormat overrides the default decoder used for structured types
// when no per-field format is provided.
func WithDefaultFormat(name string) Option {
	return func(l *Loader) {
		l.defaultFormat = strings.ToLower(name)
	}
}

// WithProviderPrefix supplies a function whose result is prepended to provider
// keys prior to lookup (for example to inject environment names).
func WithProviderPrefix(fn func() string) Option {
	return func(l *Loader) {
		l.prefixFunc = fn
	}
}

// WithProviderSuffix supplies a function whose result is appended to provider
// keys prior to lookup.
func WithProviderSuffix(fn func() string) Option {
	return func(l *Loader) {
		l.suffixFunc = fn
	}
}
