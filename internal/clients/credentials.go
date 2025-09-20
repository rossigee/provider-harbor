/*
Copyright 2024 Crossplane Harbor Provider.
*/

package clients

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
)

// GetCredentialsFromSecret retrieves credentials from a Kubernetes secret
func GetCredentialsFromSecret(ctx context.Context, k8sClient client.Client, secretRef xpv1.SecretReference) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	nn := types.NamespacedName{
		Namespace: secretRef.Namespace,
		Name:      secretRef.Name,
	}

	if err := k8sClient.Get(ctx, nn, secret); err != nil {
		return nil, err
	}

	return secret, nil
}