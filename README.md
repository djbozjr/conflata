# Conflata

[![Go Reference](https://pkg.go.dev/badge/github.com/djbozjr/conflata.svg)](https://pkg.go.dev/github.com/djbozjr/conflata)
[![Go Report Card](https://goreportcard.com/badge/github.com/djbozjr/conflata)](https://goreportcard.com/report/github.com/djbozjr/conflata)
[![CI](https://github.com/djbozentka/conflata/actions/workflows/ci.yml/badge.svg)](https://github.com/djbozentka/conflata/actions/workflows/ci.yml)
[![Coverage](https://codecov.io/gh/djbozentka/conflata/branch/main/graph/badge.svg)](https://codecov.io/gh/djbozentka/conflata)

Conflata is a Go configuration library that loads struct fields from environment variables first and falls back to external providers (AWS Secrets Manager, HashiCorp Vault, Google Secret Manager, or custom providers). It automatically decodes primitives and structured types and reports aggregated field errors so callers can decide how to handle missing values.

## Installation

```bash
go get github.com/djbozjr/conflata
```

## Quick Start

```go
type Config struct {
    DatabaseURL string        `conflata:"env:DATABASE_URL provider:database-url backend:vault"`
    PoolSize    int           `conflata:"env:DB_POOL_SIZE"`
    Timeout     time.Duration `conflata:"env:DB_TIMEOUT"`
}

loader := conflata.New(
    conflata.WithProvider("vault", vaultProvider),
    conflata.WithDefaultProvider("vault"),
)

var cfg Config
if err := loader.Load(context.Background(), &cfg); err != nil {
    if group, ok := err.(*conflata.ErrorGroup); ok {
        for _, fe := range group.Fields() {
            log.Printf("field %s failed: %v", fe.FieldPath, fe.Attempts)
        }
    } else {
        log.Fatalf("invalid target: %v", err)
    }
}
```

`Loader.Load` prefers environment variables when set. If a value is missing, it queries the configured provider and, when failures occur, returns an `*conflata.ErrorGroup` so you can inspect and handle them.

## Struct Tags

Describe how a field should be populated using the `conflata` tag:

```
conflata:"env:DATABASE_URL provider:prod/database backend:vault format:json"
```

| Key       | Description |
|-----------|-------------|
| `env`     | Environment variable to read first. |
| `provider`| Remote secret identifier (Vault path, AWS secret name, GCP secret). |
| `backend` | Provider registration name. Defaults to `aws` unless overridden with `WithDefaultProvider`. |
| `format`  | Decoder to use (`json`, `xml`, `text`, or custom formats registered via `WithDecoder`). |
| `default` | Literal fallback value used when both `env` and `provider` fail or are omitted. Quote values containing spaces, e.g. `default:"my name"` |

At least one of `env` or `provider` must be present. Environment values override provider values when both succeed.

### Advanced Usage

- **Nested structs:** Tag an entire struct field (e.g. `API APISettings "conflata:\"env:API_JSON provider:api/settings\""`) to hydrate JSON/XML payloads while still allowing nested fields to declare their own tags (e.g. `API.Token`).
- **Selective loading:** Fields without a `conflata` tag are skipped entirely—only tag the fields you want Conflata to manage.
- **Custom decoders:** Register new formats with `WithDecoder` and reference them in tags, or set a new default decoder globally with `WithDefaultFormat`.
- **Defaults:** Provide `default:"literal"` on any field to supply a fallback when env/provider values are absent.
- **Provider namespacing:** Use `WithProviderPrefix`/`WithProviderSuffix` to dynamically prepend/append identifiers (e.g., environment names) to provider keys before lookup.
- **Custom providers:** Implement the `conflata.Provider` interface and register instances via `WithProvider`.
- **Error inspection:** `Loader.Load` returns an `*ErrorGroup`. Iterate the grouped `FieldError`s to determine which configuration values failed and why without aborting the entire load.

### Decoding

Conflata automatically chooses a decoder:

- Primitives (`string`, numeric types, booleans, `time.Duration`, `[]byte`) parse from plain text.
- Structs, slices, arrays, maps, and interfaces default to JSON.
- Pointer fields are allocated as needed.

Override with the `format:` tag or global `WithDefaultFormat`.

### Errors

When `Loader.Load` returns an error, type assert it to `*conflata.ErrorGroup`. Each group exposes `Fields()` containing the attempted sources (environment, provider, decoder) per field so you can decide whether to continue with partial configuration or fail fast.

## Custom Providers

Any type implementing `Fetch(ctx context.Context, key string) (string, error)` can be registered via `WithProvider`. Built-in providers for AWS Secrets Manager, Vault KV v2, and Google Secret Manager live under `providers/`.

## Examples

The `examples/` directory contains runnable programs for AWS, GCP, and Vault. Each demonstrates nested struct loading and includes stub providers when `CONFLATA_EXAMPLE_STUB=1`.

```bash
cd examples/aws
go run .

CONFLATA_EXAMPLE_STUB=1 go run ./examples/gcp
```

## Providers

### AWS Secrets Manager

```go
smClient := secretsmanager.NewFromConfig(cfg)
awsProvider, _ := awssm.New(smClient)
loader := conflata.New(
    conflata.WithProvider("aws", awsProvider),
    conflata.WithDefaultProvider("aws"),
)
```

Environment requirements: standard AWS credential chain (env vars, shared config, metadata). Ensure IAM permissions for Secrets Manager.

### HashiCorp Vault

```go
client, _ := vaultapi.NewClient(vaultapi.DefaultConfig())
vaultProvider, _ := vault.FromClient(client, "secret")
```

Environment requirements: `VAULT_ADDR` plus a token (via `VAULT_TOKEN`, AppRole, or agent). Conflata automatically inspects KV data maps and falls back to JSON if no `value` key exists.

### Google Secret Manager

```go
client, _ := secretmanager.NewClient(ctx)
gcpProvider, _ := gcpsecret.New(client, gcpsecret.WithProject("my-project"))
```

Environment requirements: Application Default Credentials (service account JSON via `GOOGLE_APPLICATION_CREDENTIALS`, gcloud auth application-default login, or running on GCP runtimes).

## Secret Payload Formats

Structured fields default to JSON unless another format or decoder is specified.

| Field | Payload Example |
|-------|-----------------|
| `DatabaseCredentials` struct | `{"username":"app","password":"s3cr3t"}` |
| `[]string` | `["primary","replica"]` |
| `CacheSettings` with custom `kv` decoder | `enabled=true,ttl=30s` |
| `time.Duration` | `5s` |

Vault KV mounts can either store individual keys per field or a JSON blob per nested struct. See the examples for both approaches.

## Testing

```bash
GOCACHE=$(pwd)/.gocache go test ./...
```

## Development

```bash
gofmt -w .
GOCACHE=$(pwd)/.gocache go test ./...
GOCACHE=$(pwd)/.gocache go vet ./...
staticcheck ./...
```

Install Staticcheck with `go install honnef.co/go/tools/cmd/staticcheck@latest`. Releases follow semantic versioning; run `./scripts/release.sh v1.0.0` and `git push origin v1.0.0` to tag a release.
