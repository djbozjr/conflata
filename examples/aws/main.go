package main

import (
	"context"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"

	"github.com/djbozjr/conflata"
	"github.com/djbozjr/conflata/examples/internal/exampleutil"
	"github.com/djbozjr/conflata/examples/internal/stub"
	"github.com/djbozjr/conflata/providers/awssm"
)

// Database credentials are nested to demonstrate recursive traversal.
type DatabaseCredentials struct {
	Username string `conflata:"env:DB_USERNAME provider:prod/db-username"`
	Password string `conflata:"env:DB_PASSWORD provider:prod/db-password"`
}

type DatabaseConfig struct {
	URL         string              `conflata:"env:DATABASE_URL provider:prod/database-url"`
	MaxOpenConn int                 `conflata:"env:DB_MAX_OPEN provider:prod/database-max-open"`
	Creds       DatabaseCredentials // fields inside this struct carry their own tags
}

type MessagingConfig struct {
	BrokerURL string        `conflata:"env:KAFKA_URL provider:prod/kafka-url"`
	Timeout   time.Duration `conflata:"env:KAFKA_TIMEOUT provider:prod/kafka-timeout"`
}

type AppConfig struct {
	ServiceName string `conflata:"env:SERVICE_NAME provider:prod/service-name"`
	Database    DatabaseConfig
	Messaging   *MessagingConfig
}

func main() {
	ctx := context.Background()

	// In CI or when no AWS credentials are available, fall back to a stub
	// provider to keep the example runnable.
	provider := loadAWSProvider(ctx)

	loader := conflata.New(
		conflata.WithProvider("aws", provider),
		conflata.WithDefaultProvider("aws"),
	)

	var cfg AppConfig
	if err := loader.Load(ctx, &cfg); err != nil {
		if !exampleutil.ReportWarnings(err) {
			log.Fatalf("load config: %v", err)
		}
	}

	log.Printf("conflata example loaded config: %#v", cfg)
}

func loadAWSProvider(ctx context.Context) conflata.Provider {
	if stub.Enabled() {
		stub.PopulateEnv(map[string]string{
			"SERVICE_NAME":  "stub-service",
			"DATABASE_URL":  "postgres://stub-db",
			"DB_MAX_OPEN":   "50",
			"DB_USERNAME":   "env-user",
			"DB_PASSWORD":   "env-pass",
			"KAFKA_URL":     "kafka://env",
			"KAFKA_TIMEOUT": "10s",
		})
		return stub.NewProvider(map[string]string{
			"prod/db-username":       "stub-user",
			"prod/db-password":       "stub-pass",
			"prod/database-url":      "postgres://stub",
			"prod/database-max-open": "25",
			"prod/kafka-url":         "kafka://stub",
			"prod/kafka-timeout":     "5s",
			"prod/service-name":      "stub-service",
		})
	}

	awsCfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("load aws config: %v", err)
	}
	smClient := secretsmanager.NewFromConfig(awsCfg)
	p, err := awssm.New(smClient)
	if err != nil {
		log.Fatalf("create aws provider: %v", err)
	}
	return p
}
