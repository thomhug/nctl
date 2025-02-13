package get

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	iam "github.com/ninech/apis/iam/v1alpha1"
	"github.com/ninech/nctl/api"
)

type apiServiceAccountsCmd struct {
	Name            string `arg:"" help:"Name of the API Service Account to get. If omitted all in the namespace will be listed." default:""`
	PrintToken      bool   `help:"Print the bearer token of the Account. Requires name to be set." default:"false"`
	PrintKubeconfig bool   `help:"Print the kubeconfig of the Account. Requires name to be set." default:"false"`
}

const (
	tokenKey      = "token"
	kubeconfigKey = "kubeconfig"
)

func (asa *apiServiceAccountsCmd) Run(ctx context.Context, client *api.Client, get *Cmd) error {
	header := get.Output == full

	if len(asa.Name) != 0 {
		sa := &iam.APIServiceAccount{}
		if err := client.Get(ctx, client.Name(asa.Name), sa); err != nil {
			return fmt.Errorf("unable to get API Service Account %s: %w", asa.Name, err)
		}

		if asa.PrintToken {
			return asa.printToken(ctx, client, sa)
		}

		if asa.PrintKubeconfig {
			return asa.printKubeconfig(ctx, client, sa)
		}

		return asa.print([]iam.APIServiceAccount{*sa}, header)
	}

	if asa.PrintToken || asa.PrintKubeconfig {
		return fmt.Errorf("name is not set, token or kubeconfig can only be printed for a single API Service Account")
	}

	asaList := &iam.APIServiceAccountList{}

	if err := list(ctx, client, asaList, get.AllNamespaces); err != nil {
		return err
	}

	if len(asaList.Items) == 0 {
		printEmptyMessage(iam.APIServiceAccountKind, client.Namespace)
		return nil
	}

	return asa.print(asaList.Items, header)
}

func (asa *apiServiceAccountsCmd) print(sas []iam.APIServiceAccount, header bool) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)

	if header {
		fmt.Fprintf(w, "%s\t%s\t%s\n", "NAME", "NAMESPACE", "ROLE")
	}

	for _, sa := range sas {
		fmt.Fprintf(w, "%s\t%s\t%s\n", sa.Name, sa.Namespace, sa.Spec.ForProvider.Role)
	}

	return w.Flush()
}

func (asa *apiServiceAccountsCmd) printToken(ctx context.Context, client *api.Client, sa *iam.APIServiceAccount) error {
	secret, err := client.GetConnectionSecret(ctx, sa)
	if err != nil {
		return fmt.Errorf("unable to get connection secret: %w", err)
	}

	token, ok := secret.Data[tokenKey]
	if !ok {
		return fmt.Errorf("secret of API Service Account %s has no token", sa.Name)
	}

	fmt.Printf("%s\n", token)

	return nil
}

func (asa *apiServiceAccountsCmd) printKubeconfig(ctx context.Context, client *api.Client, sa *iam.APIServiceAccount) error {
	secret, err := client.GetConnectionSecret(ctx, sa)
	if err != nil {
		return fmt.Errorf("unable to get connection secret: %w", err)
	}

	kc, ok := secret.Data[kubeconfigKey]
	if !ok {
		return fmt.Errorf("secret of API Service Account %s has no kubeconfig", sa.Name)
	}

	fmt.Printf("%s", kc)

	return nil
}
