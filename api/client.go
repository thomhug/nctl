package api

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/ninech/apis"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type Client struct {
	runtimeclient.WithWatch
	Config         *rest.Config
	KubeconfigPath string
	Namespace      string
}

// New returns a new Client by loading a kubeconfig with the supplied context
// and namespace. The kubeconfig is discovered like this:
// * KUBECONFIG environment variable pointing at a file
// * $HOME/.kube/config if exists
func New(apiClusterContext, namespace string) (*Client, error) {
	client := &Client{
		Namespace: namespace,
	}
	if err := client.loadConfig(apiClusterContext); err != nil {
		return nil, err
	}

	scheme, err := NewScheme()
	if err != nil {
		return nil, err
	}

	mapper := apis.StaticRESTMapper(scheme)
	mapper.Add(corev1.SchemeGroupVersion.WithKind("Secret"), meta.RESTScopeNamespace)

	c, err := runtimeclient.NewWithWatch(client.Config, runtimeclient.Options{
		Scheme: scheme,
		Mapper: mapper,
	})
	if err != nil {
		return nil, err
	}

	client.WithWatch = c
	return client, nil
}

// NewScheme returns a *runtime.Scheme with all the relevant types registered.
func NewScheme() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	if err := apis.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := corev1.AddToScheme(scheme); err != nil {
		return nil, err
	}
	return scheme, nil
}

// adapted from https://github.com/kubernetes-sigs/controller-runtime/blob/4c9c9564e4652bbdec14a602d6196d8622500b51/pkg/client/config/config.go#L116
func (c *Client) loadConfig(context string) error {
	loadingRules, err := LoadingRules()
	if err != nil {
		return err
	}

	cfg, namespace, err := loadConfigWithContext("", loadingRules, context)
	if err != nil {
		return err
	}
	if c.Namespace == "" {
		c.Namespace = namespace
	}
	c.Config = cfg
	c.KubeconfigPath = loadingRules.GetDefaultFilename()

	return nil
}

func (c *Client) Name(name string) types.NamespacedName {
	return types.NamespacedName{Name: name, Namespace: c.Namespace}
}

func (c *Client) GetConnectionSecret(ctx context.Context, mg resource.Managed) (*corev1.Secret, error) {
	if mg.GetWriteConnectionSecretToReference() == nil {
		return nil, fmt.Errorf("%T %s/%s has no connection secret ref set", mg, mg.GetName(), mg.GetNamespace())
	}

	nsName := types.NamespacedName{
		Name:      mg.GetWriteConnectionSecretToReference().Name,
		Namespace: mg.GetWriteConnectionSecretToReference().Namespace,
	}
	secret := &corev1.Secret{}
	if err := c.Get(ctx, nsName, secret); err != nil {
		return nil, fmt.Errorf("unable to get referenced secret %v: %w", nsName, err)
	}

	return secret, nil
}

func LoadingRules() (*clientcmd.ClientConfigLoadingRules, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	if _, ok := os.LookupEnv("HOME"); !ok {
		u, err := user.Current()
		if err != nil {
			return nil, fmt.Errorf("could not get current user: %w", err)
		}
		loadingRules.Precedence = append(
			loadingRules.Precedence,
			filepath.Join(u.HomeDir, clientcmd.RecommendedHomeDir, clientcmd.RecommendedFileName),
		)
	}

	return loadingRules, nil
}

func loadConfigWithContext(apiServerURL string, loader clientcmd.ClientConfigLoader, context string) (*rest.Config, string, error) {
	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loader, &clientcmd.ConfigOverrides{
			ClusterInfo: clientcmdapi.Cluster{
				Server: apiServerURL,
			},
			CurrentContext: context,
		},
	)

	ns, _, err := clientConfig.Namespace()
	if err != nil {
		return nil, "", err
	}

	cfg, err := clientConfig.ClientConfig()
	return cfg, ns, err
}

func ObjectName(obj runtimeclient.Object) types.NamespacedName {
	return types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()}
}
