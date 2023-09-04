package utils

import (
	"context"
	"github.com/GDATASoftwareAG/cluster-api-provider-ionoscloud/api/v1alpha1"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

const (
	UserNameProperty = "username"
	PasswordProperty = "password"
	TokenProperty    = "token"
	defaultNamespace = "capic-system"
)

func GetLoginDataForCluster(ctx context.Context, client client.Client, ionoscloudCluster *v1alpha1.IONOSCloudCluster) (username, password, token, host string, err error) {
	ionoscloudClusterIdentity := &v1alpha1.IONOSCloudClusterIdentity{}
	nsn := types.NamespacedName{
		Name: ionoscloudCluster.Spec.IdentityName,
	}
	err = client.Get(ctx, nsn, ionoscloudClusterIdentity)
	if err != nil {
		return "", "", "", "", errors.Wrapf(err, "failed to get identity")
	}
	return GetLoginDataFromIdentity(ctx, client, ionoscloudClusterIdentity)
}

func GetLoginDataFromIdentity(ctx context.Context, client client.Client, ionoscloudClusterIdentity *v1alpha1.IONOSCloudClusterIdentity) (username, password, token, host string, err error) {
	host = ionoscloudClusterIdentity.Spec.HostUrl
	secret := &v1.Secret{}
	nsn := types.NamespacedName{Name: ionoscloudClusterIdentity.Spec.SecretName, Namespace: IdentitySecretNamespace()}
	if err := client.Get(ctx, nsn, secret); err != nil {
		return "", "", "", "", errors.Wrapf(err, "failed to get secret")
	}

	if u, ok := secret.Data[UserNameProperty]; ok {
		username = string(u)
	}

	if p, ok := secret.Data[PasswordProperty]; ok {
		password = string(p)
	}

	if t, ok := secret.Data[TokenProperty]; ok {
		token = string(t)
	}

	if len(token) > 0 && len(username) == 0 && len(password) == 0 || len(token) == 0 && len(username) > 0 && len(password) > 0 {
		return username, password, token, host, nil
	} else {
		return "", "", "", "", errors.New("either username and password or token must be specified")
	}
}

func IdentitySecretNamespace() string {
	if ns, ok := os.LookupEnv("POD_NAMESPACE"); ok {
		return ns
	} else if data, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
		if ns := strings.TrimSpace(string(data)); len(ns) > 0 {
			return ns
		}
	}
	return defaultNamespace
}
