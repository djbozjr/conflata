// Package conflata loads configuration structs by reading environment variables
// first and falling back to external secret providers such as AWS Secrets
// Manager, HashiCorp Vault, or Google Secret Manager. Fields are annotated with
// `conflata` struct tags that describe which env/providor keys to read, and the
// loader reports grouped errors so callers can decide how to handle missing
// values.
//
// Example:
//
//	type Config struct {
//	    DatabaseURL string `conflata:"env:DATABASE_URL provider:prod/database-url"`
//	}
//
//	loader := conflata.New(conflata.WithProvider("aws", awsProvider))
//	if errGroup, err := loader.Load(ctx, &cfg); errGroup != nil {
//	    log.Println(errGroup)
//	}
package conflata
