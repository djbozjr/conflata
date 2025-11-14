package awssm

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

type stubClient struct {
	input *secretsmanager.GetSecretValueInput
	out   *secretsmanager.GetSecretValueOutput
	err   error
}

func (s *stubClient) GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
	s.input = params
	if s.err != nil {
		return nil, s.err
	}
	return s.out, nil
}

func TestProviderFetchString(t *testing.T) {
	stub := &stubClient{
		out: &secretsmanager.GetSecretValueOutput{
			SecretString: aws.String("value"),
		},
	}
	provider, err := New(stub, WithVersionStage("AWSCURRENT"))
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	got, err := provider.Fetch(context.Background(), "secret")
	if err != nil {
		t.Fatalf("Fetch error: %v", err)
	}
	if got != "value" {
		t.Fatalf("expected value, got %s", got)
	}
	if stub.input == nil || aws.ToString(stub.input.VersionStage) != "AWSCURRENT" {
		t.Fatalf("expected version stage to be set, got %+v", stub.input)
	}
}

func TestProviderFetchBinary(t *testing.T) {
	stub := &stubClient{
		out: &secretsmanager.GetSecretValueOutput{
			SecretBinary: []byte("abc"),
		},
	}
	provider, err := New(stub)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	got, err := provider.Fetch(context.Background(), "secret")
	if err != nil {
		t.Fatalf("Fetch error: %v", err)
	}
	if got != "abc" {
		t.Fatalf("expected binary payload as string, got %s", got)
	}
}

func TestProviderFetchMissingPayload(t *testing.T) {
	stub := &stubClient{
		out: &secretsmanager.GetSecretValueOutput{},
	}
	provider, err := New(stub)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if _, err := provider.Fetch(context.Background(), "secret"); err == nil {
		t.Fatal("expected error for missing payload")
	}
}

func TestNewRequiresClient(t *testing.T) {
	if _, err := New(nil); err == nil {
		t.Fatal("expected error when client is nil")
	}
}

func TestProviderPropagatesError(t *testing.T) {
	stub := &stubClient{
		err: errors.New("boom"),
	}
	provider, err := New(stub)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if _, err := provider.Fetch(context.Background(), "secret"); err == nil {
		t.Fatal("expected error")
	}
}
