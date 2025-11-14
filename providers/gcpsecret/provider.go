package gcpsecret

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/googleapis/gax-go/v2"
)

// Client represents the subset of the GCP Secret Manager client used.
type Client interface {
	AccessSecretVersion(ctx context.Context, req *secretmanagerpb.AccessSecretVersionRequest, opts ...gax.CallOption) (*secretmanagerpb.AccessSecretVersionResponse, error)
}

// Provider fetches secrets from Google Secret Manager.
type Provider struct {
	client  Client
	project string
	version string
}

// Option configures the provider.
type Option func(*Provider)

// WithProject sets the default project ID used when tags reference short secret
// names instead of fully qualified resource names.
func WithProject(projectID string) Option {
	return func(p *Provider) {
		p.project = projectID
	}
}

// WithVersion overrides the default version (latest).
func WithVersion(version string) Option {
	return func(p *Provider) {
		if version != "" {
			p.version = version
		}
	}
}

// New constructs a Secret Manager provider.
func New(client Client, opts ...Option) (*Provider, error) {
	if client == nil {
		return nil, errors.New("gcpsecret: client is required")
	}
	p := &Provider{
		client:  client,
		version: "latest",
	}
	for _, opt := range opts {
		opt(p)
	}
	return p, nil
}

// Fetch retrieves the secret identified by key. Keys can either be full resource
// names (projects/*/secrets/*/versions/*) or shorthand secret IDs when a project
// was provided via options.
func (p *Provider) Fetch(ctx context.Context, key string) (string, error) {
	if key == "" {
		return "", errors.New("gcpsecret: secret name cannot be empty")
	}
	name := key
	if !strings.HasPrefix(key, "projects/") {
		if p.project == "" {
			return "", errors.New("gcpsecret: project must be set when using short secret names")
		}
		name = fmt.Sprintf("projects/%s/secrets/%s/versions/%s", p.project, key, p.version)
	}
	resp, err := p.client.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{Name: name})
	if err != nil {
		return "", fmt.Errorf("gcpsecret: %w", err)
	}
	if resp.GetPayload() == nil || len(resp.Payload.Data) == 0 {
		return "", errors.New("gcpsecret: secret payload empty")
	}
	return string(resp.Payload.Data), nil
}
