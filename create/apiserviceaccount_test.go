package create

import (
	"context"
	"testing"
	"time"

	"github.com/ninech/nctl/api"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestAPIServiceAccount(t *testing.T) {
	scheme, err := api.NewScheme()
	if err != nil {
		t.Fatal(err)
	}

	cmd := apiServiceAccountCmd{
		Wait:        false,
		WaitTimeout: time.Second,
	}

	asa := cmd.newAPIServiceAccount("default")
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(asa).Build()
	apiClient := &api.Client{WithWatch: client, Namespace: "default"}
	ctx := context.Background()

	if err := cmd.Run(ctx, apiClient); err != nil {
		t.Fatal(err)
	}

	if err := apiClient.Get(ctx, api.ObjectName(asa), asa); err != nil {
		t.Fatalf("expected asa to exist, got: %s", err)
	}
}
