package main

import (
	"context"
	"log"

	vaultapi "github.com/hashicorp/vault/api"

	"github.com/djbozjr/conflata"
	"github.com/djbozjr/conflata/examples/internal/exampleutil"
	"github.com/djbozjr/conflata/examples/internal/stub"
	"github.com/djbozjr/conflata/providers/vault"
)

type TLSConfig struct {
	CertPEM string `conflata:"env:TLS_CERT provider:secret/data/tls-cert"`
	KeyPEM  string `conflata:"env:TLS_KEY provider:secret/data/tls-key"`
}

type Metadata struct {
	Owner  string `conflata:"env:SERVER_OWNER provider:secret/data/server-owner"`
	Region string `conflata:"env:SERVER_REGION provider:secret/data/server-region"`
}

type ServerConfig struct {
	Host     string `conflata:"env:SERVER_HOST provider:secret/data/server-host"`
	Port     int    `conflata:"env:SERVER_PORT provider:secret/data/server-port"`
	TLS      TLSConfig
	Metadata Metadata
}

func main() {
	ctx := context.Background()

	provider := loadVaultProvider()

	loader := conflata.New(
		conflata.WithProvider("vault", provider),
		conflata.WithDefaultProvider("vault"),
	)

	var cfg ServerConfig
	errGroup, err := loader.Load(ctx, &cfg)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	exampleutil.ReportWarnings(errGroup)
	log.Printf("server configuration: %#v", cfg)
}

func loadVaultProvider() conflata.Provider {
	if stub.Enabled() {
		stub.PopulateEnv(map[string]string{
			"TLS_CERT":      "-----BEGIN CERTIFICATE-----",
			"TLS_KEY":       "-----BEGIN PRIVATE KEY-----",
			"SERVER_OWNER":  "ops-team@example.com",
			"SERVER_REGION": "us-west-2",
			"SERVER_HOST":   "0.0.0.0",
			"SERVER_PORT":   "8443",
		})
		return stub.NewProvider(map[string]string{
			"secret/data/tls-cert":      "-----BEGIN CERTIFICATE-----",
			"secret/data/tls-key":       "-----BEGIN PRIVATE KEY-----",
			"secret/data/server-owner":  "ops-team@example.com",
			"secret/data/server-region": "us-west-2",
			"secret/data/server-host":   "0.0.0.0",
			"secret/data/server-port":   "8443",
		})
	}

	client, err := vaultapi.NewClient(vaultapi.DefaultConfig())
	if err != nil {
		log.Fatalf("create vault client: %v", err)
	}
	provider, err := vault.FromClient(client, "secret")
	if err != nil {
		log.Fatalf("create vault provider: %v", err)
	}
	return provider
}
