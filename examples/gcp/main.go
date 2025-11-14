package main

import (
	"context"
	"errors"
	"log"
	"time"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/googleapis/gax-go/v2"

	"github.com/djbozjr/conflata"
	"github.com/djbozjr/conflata/examples/internal/exampleutil"
	"github.com/djbozjr/conflata/examples/internal/stub"
	"github.com/djbozjr/conflata/providers/gcpsecret"
)

type CacheSettings struct {
	Enabled bool          `json:"enabled"`
	TTL     time.Duration `json:"ttl"`
}

type APISettings struct {
	BaseURL string        `json:"baseUrl"`
	Timeout time.Duration `json:"timeout"`
	Cache   CacheSettings `json:"cache"`
	Token   string        `conflata:"env:API_TOKEN provider:api/token"`
}

type Config struct {
	ProjectID string      `conflata:"env:GCP_PROJECT_ID provider:projects/my-project/secrets/project-id/versions/latest"`
	API       APISettings `conflata:"env:API_SETTINGS provider:api/settings"`
}

func main() {
	ctx := context.Background()

	provider, cleanup := loadGCPProvider(ctx)
	defer cleanup()

	loader := conflata.New(
		conflata.WithProvider("gcp", provider),
		conflata.WithDefaultProvider("gcp"),
	)

	var cfg Config
	if err := loader.Load(ctx, &cfg); err != nil {
		if !exampleutil.ReportWarnings(err) {
			log.Fatalf("load config: %v", err)
		}
	}
	log.Printf("loaded config: %#v", cfg)
}

type stubGCPClient struct {
	secrets map[string]string
}

func (s stubGCPClient) AccessSecretVersion(ctx context.Context, req *secretmanagerpb.AccessSecretVersionRequest, _ ...gax.CallOption) (*secretmanagerpb.AccessSecretVersionResponse, error) {
	if value, ok := s.secrets[req.GetName()]; ok {
		return &secretmanagerpb.AccessSecretVersionResponse{
			Payload: &secretmanagerpb.SecretPayload{Data: []byte(value)},
		}, nil
	}
	return nil, errors.New("missing secret")
}

func loadGCPProvider(ctx context.Context) (conflata.Provider, func()) {
	if stub.Enabled() {
		stub.PopulateEnv(map[string]string{
			"GCP_PROJECT_ID": "stub-project",
			"API_SETTINGS":   `{"baseUrl":"https://env-api","timeout":12000000000,"cache":{"enabled":true,"ttl":20000000000}}`,
			"API_TOKEN":      "env-token",
		})
		client := stubGCPClient{
			secrets: map[string]string{
				"projects/my-gcp-project/secrets/project-id/versions/latest": "stub-project",
				"api/settings": `{"baseUrl":"https://stub-api","timeout":15000000000,"cache":{"enabled":true,"ttl":45000000000}}`,
				"api/token":    "provider-token",
			},
		}
		p, _ := gcpsecret.New(client, gcpsecret.WithProject("my-gcp-project"))
		return p, func() {}
	}

	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		log.Fatalf("create secret manager client: %v", err)
	}
	provider, err := gcpsecret.New(client, gcpsecret.WithProject("my-gcp-project"))
	if err != nil {
		log.Fatalf("create gcp provider: %v", err)
	}
	return provider, func() {
		_ = client.Close()
	}
}
