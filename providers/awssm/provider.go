package awssm

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

// SecretsManagerClient captures the subset of the AWS Secrets Manager client
// used by the provider. *secretsmanager.Client satisfies this interface.
type SecretsManagerClient interface {
	GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
}

// Provider loads values from AWS Secrets Manager.
type Provider struct {
	client       SecretsManagerClient
	versionStage *string
	versionID    *string
	callOpts     []func(*secretsmanager.Options)
}

// Option configures the AWS provider.
type Option func(*Provider)

// WithVersionStage requests a specific version stage (defaults to AWS Current).
func WithVersionStage(stage string) Option {
	return func(p *Provider) {
		if stage != "" {
			p.versionStage = aws.String(stage)
		}
	}
}

// WithVersionID requests a specific version ID.
func WithVersionID(id string) Option {
	return func(p *Provider) {
		if id != "" {
			p.versionID = aws.String(id)
		}
	}
}

// WithClientOptions forwards Secrets Manager call options to each fetch.
func WithClientOptions(opts ...func(*secretsmanager.Options)) Option {
	return func(p *Provider) {
		p.callOpts = append(p.callOpts, opts...)
	}
}

// New constructs a Secrets Manager provider.
func New(client SecretsManagerClient, opts ...Option) (*Provider, error) {
	if client == nil {
		return nil, errors.New("awssm: client is required")
	}
	p := &Provider{
		client: client,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p, nil
}

// Fetch retrieves the secret with the provided key.
func (p *Provider) Fetch(ctx context.Context, key string) (string, error) {
	if key == "" {
		return "", errors.New("awssm: secret id cannot be empty")
	}
	input := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(key),
	}
	if p.versionStage != nil {
		input.VersionStage = p.versionStage
	}
	if p.versionID != nil {
		input.VersionId = p.versionID
	}
	out, err := p.client.GetSecretValue(ctx, input, p.callOpts...)
	if err != nil {
		return "", fmt.Errorf("awssm: %w", err)
	}
	if out.SecretString != nil {
		return aws.ToString(out.SecretString), nil
	}
	if len(out.SecretBinary) > 0 {
		return string(out.SecretBinary), nil
	}
	return "", errors.New("awssm: secret contained no payload")
}
