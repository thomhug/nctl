package auth

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"strings"

	infrastructure "github.com/ninech/apis/infrastructure/v1alpha1"
	"github.com/ninech/nctl/api"
	"k8s.io/apimachinery/pkg/types"
)

type ClusterCmd struct {
	Name       string `arg:"" help:"Name of the cluster to authenticate with. Also accepts 'name/namespace' format."`
	ExecPlugin bool   `help:"Automatically run exec plugin after writing the kubeconfig."`
}

func (a *ClusterCmd) Run(ctx context.Context, client *api.Client) error {
	name, err := clusterName(a.Name, client.Namespace)
	if err != nil {
		return err
	}

	cluster := &infrastructure.KubernetesCluster{}
	if err := client.Get(ctx, name, cluster); err != nil {
		return err
	}

	apiEndpoint, err := url.Parse(cluster.Status.AtProvider.APIEndpoint)
	if err != nil {
		return fmt.Errorf("invalid cluster API endpoint: %w", err)
	}

	issuerURL, err := url.Parse(cluster.Status.AtProvider.OIDCIssuerURL)
	if err != nil {
		return fmt.Errorf("invalid cluster OIDC issuer url: %w", err)
	}

	caCert, err := base64.StdEncoding.DecodeString(cluster.Status.AtProvider.APICACert)
	if err != nil {
		return fmt.Errorf("unable to decode API CA certificate: %w", err)
	}

	// not sure if this should ever happen but better than getting a panic
	if len(os.Args) == 0 {
		return fmt.Errorf("could not get command name from os.Args")
	}
	command := os.Args[0]

	cfg, err := newAPIConfig(
		apiEndpoint,
		issuerURL,
		command,
		cluster.Status.AtProvider.OIDCClientID,
		overrideName(ContextName(cluster)),
		setCACert(caCert),
	)
	if err != nil {
		return fmt.Errorf("unable to create kubeconfig: %w", err)
	}

	if err := login(cfg, client.KubeconfigPath, runExecPlugin(a.ExecPlugin), switchCurrentContext()); err != nil {
		return fmt.Errorf("error logging in to cluster %s: %w", name, err)
	}

	return nil
}

func clusterName(name, namespace string) (types.NamespacedName, error) {
	parts := strings.Split(name, "/")
	if len(parts) == 2 {
		name = parts[0]
		namespace = parts[1]
	}

	if namespace == "" {
		return types.NamespacedName{}, fmt.Errorf("namespace cannot be empty")
	}

	return types.NamespacedName{Name: name, Namespace: namespace}, nil
}

func ContextName(cluster *infrastructure.KubernetesCluster) string {
	return fmt.Sprintf("%s/%s", cluster.Name, cluster.Namespace)
}
