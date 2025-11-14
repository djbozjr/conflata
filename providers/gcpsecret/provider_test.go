package gcpsecret

import (
	"context"
	"errors"
	"testing"

	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/googleapis/gax-go/v2"
)

type stubClient struct {
	lastRequest *secretmanagerpb.AccessSecretVersionRequest
	response    *secretmanagerpb.AccessSecretVersionResponse
	err         error
}

func (s *stubClient) AccessSecretVersion(ctx context.Context, req *secretmanagerpb.AccessSecretVersionRequest, _ ...gax.CallOption) (*secretmanagerpb.AccessSecretVersionResponse, error) {
	s.lastRequest = req
	if s.err != nil {
		return nil, s.err
	}
	return s.response, nil
}

func TestProviderFetchShortName(t *testing.T) {
	stub := &stubClient{
		response: &secretmanagerpb.AccessSecretVersionResponse{
			Payload: &secretmanagerpb.SecretPayload{Data: []byte("value")},
		},
	}
	provider, err := New(stub, WithProject("demo"), WithVersion("5"))
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	got, err := provider.Fetch(context.Background(), "db-password")
	if err != nil {
		t.Fatalf("Fetch error: %v", err)
	}
	if got != "value" {
		t.Fatalf("expected value, got %s", got)
	}
	if stub.lastRequest == nil || stub.lastRequest.Name != "projects/demo/secrets/db-password/versions/5" {
		t.Fatalf("unexpected request: %+v", stub.lastRequest)
	}
}

func TestProviderFetchFullName(t *testing.T) {
	stub := &stubClient{
		response: &secretmanagerpb.AccessSecretVersionResponse{
			Payload: &secretmanagerpb.SecretPayload{Data: []byte("value")},
		},
	}
	provider, err := New(stub)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	name := "projects/demo/secrets/db/versions/latest"
	if _, err := provider.Fetch(context.Background(), name); err != nil {
		t.Fatalf("Fetch error: %v", err)
	}
	if stub.lastRequest == nil || stub.lastRequest.Name != name {
		t.Fatalf("unexpected request name %s", stub.lastRequest.GetName())
	}
}

func TestProviderRequiresProjectForShortName(t *testing.T) {
	stub := &stubClient{}
	provider, err := New(stub)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if _, err := provider.Fetch(context.Background(), "db-password"); err == nil {
		t.Fatal("expected error when project missing")
	}
}

func TestProviderPropagatesError(t *testing.T) {
	stub := &stubClient{err: errors.New("boom")}
	provider, err := New(stub, WithProject("demo"))
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if _, err := provider.Fetch(context.Background(), "db"); err == nil {
		t.Fatal("expected fetch error")
	}
}

func TestProviderMissingPayload(t *testing.T) {
	stub := &stubClient{
		response: &secretmanagerpb.AccessSecretVersionResponse{},
	}
	provider, err := New(stub, WithProject("demo"))
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if _, err := provider.Fetch(context.Background(), "db"); err == nil {
		t.Fatal("expected payload error")
	}
}
