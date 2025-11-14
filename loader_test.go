package conflata

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

type stubProvider struct {
	values map[string]providerResponse
}

type providerResponse struct {
	value string
	err   error
}

func (s stubProvider) Fetch(ctx context.Context, key string) (string, error) {
	if resp, ok := s.values[key]; ok {
		if resp.err != nil {
			return "", resp.err
		}
		return resp.value, nil
	}
	return "", errors.New("missing secret")
}

func TestLoaderEnvPrecedence(t *testing.T) {
	type Config struct {
		DatabaseURL string `conflata:"env:DATABASE_URL provider:db-url"`
	}
	env := func(k string) (string, bool) {
		if k == "DATABASE_URL" {
			return "postgres://env", true
		}
		return "", false
	}
	loader := New(
		WithEnvLookup(env),
		WithProvider("vault", stubProvider{values: map[string]providerResponse{
			"db-url": {value: "postgres://provider"},
		}}),
		WithDefaultProvider("vault"),
	)

	var cfg Config
	group, err := loader.Load(context.Background(), &cfg)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if group != nil {
		t.Fatalf("expected no error group, got %v", group)
	}
	if cfg.DatabaseURL != "postgres://env" {
		t.Fatalf("expected env to win, got %s", cfg.DatabaseURL)
	}
}

func TestLoaderProviderFallback(t *testing.T) {
	type Config struct {
		APIKey string `conflata:"env:API_KEY provider:api-key"`
	}
	loader := New(
		WithEnvLookup(func(string) (string, bool) { return "", false }),
		WithProvider("vault", stubProvider{values: map[string]providerResponse{
			"api-key": {value: "secret"},
		}}),
		WithDefaultProvider("vault"),
	)
	var cfg Config
	group, err := loader.Load(context.Background(), &cfg)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if group != nil {
		t.Fatalf("expected success, got %v", group)
	}
	if cfg.APIKey != "secret" {
		t.Fatalf("expected provider value, got %s", cfg.APIKey)
	}
}

func TestLoaderAggregatesErrors(t *testing.T) {
	type Config struct {
		Token string `conflata:"env:TOKEN provider:token"`
	}
	provider := stubProvider{values: map[string]providerResponse{
		"token": {err: errors.New("boom")},
	}}
	loader := New(
		WithEnvLookup(func(string) (string, bool) { return "", false }),
		WithProvider("vault", provider),
		WithDefaultProvider("vault"),
	)
	var cfg Config
	group, err := loader.Load(context.Background(), &cfg)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if group == nil || len(group.Fields()) != 1 {
		t.Fatalf("expected one field error, got %v", group)
	}
	fieldErr := group.Fields()[0]
	if fieldErr.FieldPath != "Token" {
		t.Fatalf("expected field path Token, got %s", fieldErr.FieldPath)
	}
	if len(fieldErr.Attempts) != 2 {
		t.Fatalf("expected two attempts, got %d", len(fieldErr.Attempts))
	}
}

func TestLoaderDecodesStructAndPointer(t *testing.T) {
	type Nested struct {
		Enabled bool `json:"enabled"`
	}
	type Config struct {
		TimeoutSeconds int     `conflata:"env:TIMEOUT"`
		Nested         Nested  `conflata:"env:NESTED_JSON"`
		PtrNested      *Nested `conflata:"env:PTR_NESTED"`
	}
	env := map[string]string{
		"TIMEOUT":     "42",
		"NESTED_JSON": `{"enabled":true}`,
		"PTR_NESTED":  `{"enabled":false}`,
	}
	loader := New(WithEnvLookup(func(key string) (string, bool) {
		v, ok := env[key]
		return v, ok
	}))
	var cfg Config
	group, err := loader.Load(context.Background(), &cfg)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if group != nil {
		t.Fatalf("expected success, got %v", group)
	}
	if cfg.TimeoutSeconds != 42 || !cfg.Nested.Enabled || cfg.PtrNested == nil || cfg.PtrNested.Enabled {
		t.Fatalf("unexpected values: %+v", cfg)
	}
}

func TestLoaderSupportsXMLFormat(t *testing.T) {
	type XMLConfig struct {
		Value string `xml:"value"`
	}
	type Config struct {
		Data XMLConfig `conflata:"env:XML_CONF format:xml"`
	}
	envLookup := func(key string) (string, bool) {
		if key == "XML_CONF" {
			return `<XMLConfig><value>hello</value></XMLConfig>`, true
		}
		return "", false
	}
	loader := New(WithEnvLookup(envLookup))
	var cfg Config
	group, err := loader.Load(context.Background(), &cfg)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if group != nil {
		t.Fatalf("expected success, got %v", group)
	}
	if cfg.Data.Value != "hello" {
		t.Fatalf("expected xml decode, got %+v", cfg.Data)
	}
}

func TestLoaderDefaultProviderIsAWS(t *testing.T) {
	type Config struct {
		Secret string `conflata:"provider:api/secret"`
	}
	loader := New(
		WithEnvLookup(func(string) (string, bool) { return "", false }),
		WithProvider("aws", stubProvider{values: map[string]providerResponse{
			"api/secret": {value: "from-aws"},
		}}),
	)
	var cfg Config
	errGroup, err := loader.Load(context.Background(), &cfg)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if errGroup != nil {
		t.Fatalf("expected success, got %v", errGroup)
	}
	if cfg.Secret != "from-aws" {
		t.Fatalf("expected default provider to be aws, got %s", cfg.Secret)
	}
}

func TestLoaderCustomDefaultFormat(t *testing.T) {
	type Config struct {
		Settings map[string]string `conflata:"env:SETTINGS"`
	}
	env := map[string]string{
		"SETTINGS": "alpha=beta,gamma=delta",
	}
	kvDecoder := func(raw string, targetType reflect.Type) (any, error) {
		result := reflect.MakeMap(targetType)
		for _, pair := range strings.Split(raw, ",") {
			parts := strings.SplitN(pair, "=", 2)
			if len(parts) != 2 {
				continue
			}
			result.SetMapIndex(reflect.ValueOf(parts[0]), reflect.ValueOf(parts[1]))
		}
		return result.Interface(), nil
	}
	loader := New(
		WithEnvLookup(func(key string) (string, bool) {
			v, ok := env[key]
			return v, ok
		}),
		WithDecoder("kv", kvDecoder),
		WithDefaultFormat("kv"),
	)
	var cfg Config
	errGroup, err := loader.Load(context.Background(), &cfg)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if errGroup != nil {
		t.Fatalf("expected success, got %v", errGroup)
	}
	if cfg.Settings["alpha"] != "beta" || cfg.Settings["gamma"] != "delta" {
		t.Fatalf("unexpected settings %+v", cfg.Settings)
	}
}

func TestLoaderNestedStructTagDecodesAndOverrides(t *testing.T) {
	type CacheSettings struct {
		Enabled bool `json:"enabled"`
	}
	type APISettings struct {
		BaseURL string        `json:"baseUrl"`
		Timeout time.Duration `json:"timeout"`
		Cache   CacheSettings `json:"cache"`
		Token   string        `conflata:"env:API_TOKEN"`
	}
	type Config struct {
		API APISettings `conflata:"env:API_JSON"`
	}
	env := map[string]string{
		"API_JSON":  `{"baseUrl":"https://api.example","timeout":15000000000,"cache":{"enabled":true}}`,
		"API_TOKEN": "secret-token",
	}
	loader := New(WithEnvLookup(func(key string) (string, bool) {
		v, ok := env[key]
		return v, ok
	}))
	var cfg Config
	errGroup, err := loader.Load(context.Background(), &cfg)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if errGroup != nil {
		t.Fatalf("expected success, got %v", errGroup)
	}
	if cfg.API.BaseURL != "https://api.example" || cfg.API.Cache.Enabled != true {
		t.Fatalf("json decode failed: %+v", cfg.API)
	}
	if cfg.API.Token != "secret-token" {
		t.Fatalf("nested token override failed, got %s", cfg.API.Token)
	}
}

func TestLoaderNestedStructFromProvider(t *testing.T) {
	type APISettings struct {
		BaseURL string `json:"baseUrl"`
		Token   string `conflata:"env:API_TOKEN provider:api-token"`
	}
	type Config struct {
		API APISettings `conflata:"provider:api/config"`
	}
	provider := stubProvider{values: map[string]providerResponse{
		"api/config": {value: `{"baseUrl":"https://api.provider"}`},
		"api-token":  {value: "from-provider"},
	}}
	loader := New(
		WithEnvLookup(func(string) (string, bool) { return "", false }),
		WithProvider("aws", provider),
	)
	var cfg Config
	errGroup, err := loader.Load(context.Background(), &cfg)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if errGroup != nil {
		t.Fatalf("expected success, got %v", errGroup)
	}
	if cfg.API.BaseURL != "https://api.provider" || cfg.API.Token != "from-provider" {
		t.Fatalf("nested provider decode failed: %+v", cfg.API)
	}
}
