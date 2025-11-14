package conflata

import (
	"context"
	"errors"
	"strings"
)

type envSource struct {
	key    string
	lookup EnvLookupFunc
}

func (e envSource) Source() ValueSource {
	return SourceEnv
}

func (e envSource) Identifier() string {
	return e.key
}

func (e envSource) Fetch(context.Context) (string, error) {
	if value, ok := e.lookup(e.key); ok {
		return value, nil
	}
	return "", errors.New("not set")
}

type providerSource struct {
	identifier string
	fetchFunc  func(context.Context) (string, error)
}

func (p providerSource) Source() ValueSource {
	return SourceProvider
}

func (p providerSource) Identifier() string {
	return p.identifier
}

func (p providerSource) Fetch(ctx context.Context) (string, error) {
	return p.fetchFunc(ctx)
}

func (l *Loader) sourcesFor(tag fieldTag) []valueSource {
	var sources []valueSource
	if tag.EnvKey != "" {
		sources = append(sources, envSource{
			key:    tag.EnvKey,
			lookup: l.envLookup,
		})
	}
	if tag.ProviderKey != "" {
		sources = append(sources, l.newProviderSource(tag))
	}
	return sources
}

func (l *Loader) newProviderSource(tag fieldTag) valueSource {
	backendName := tag.BackendName
	if backendName == "" {
		backendName = l.defaultProvider
	}
	identifier := backendName
	if identifier == "" {
		identifier = "(default)"
	}
	provider := l.providers[strings.ToLower(backendName)]
	if provider == nil {
		return providerSource{
			identifier: identifier,
			fetchFunc: func(context.Context) (string, error) {
				return "", errors.New("provider not registered")
			},
		}
	}
	fullIdentifier := identifier + ":" + tag.ProviderKey
	return providerSource{
		identifier: fullIdentifier,
		fetchFunc: func(ctx context.Context) (string, error) {
			raw, err := provider.Fetch(ctx, tag.ProviderKey)
			if err != nil {
				return "", err
			}
			if raw == "" {
				return "", errors.New("empty secret")
			}
			return raw, nil
		},
	}
}
