## Conflata Examples

The `examples/` directory contains small programs that show how to wire Conflata with common secret providers. Each example exercises nested configuration structures so you can see how Conflata walks arbitrarily deep graphs.

### AWS Secrets Manager (`examples/aws`)

Loads a nested `AppConfig` structure from environment variables and AWS Secrets Manager using the `providers/awssm` helper. Replace the secret identifiers with your own paths.

### Google Secret Manager (`examples/gcp`)

Demonstrates using the GCP provider with a project-scoped secret. The `Config.API` field is tagged so the entire `APISettings` struct is hydrated from a JSON secret while `API.Token` still pulls from its own env/provider sources.

### HashiCorp Vault (`examples/vault`)

Shows how to connect to a Vault KV v2 mount and hydrate nested structs (`TLSConfig`, `Metadata`) from individual secrets.

Run any example via:

```bash
cd examples/aws
go run .
```

Ensure the necessary SDK authentication environment variables are in place before executing.

If you want to run the examples without cloud accounts, set `CONFLATA_EXAMPLE_STUB=1` before running `go run`. This swaps in in-memory stub providers populated with sample secrets and environment variables.
