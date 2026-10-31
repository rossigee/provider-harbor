/*
Copyright 2026 Crossplane Harbor Provider.
*/

package registry

import (
	"context"
	"strings"
	"testing"

	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/rossigee/provider-harbor/apis/registry/v1beta1"
)

// secretGetter serves a fixed set of Secrets from Get. Any other method is
// unused by getSecretFromRef and panics if reached.
type secretGetter struct {
	client.Client
	secrets map[client.ObjectKey]*corev1.Secret
}

func (g *secretGetter) Get(_ context.Context, key client.ObjectKey, obj client.Object, _ ...client.GetOption) error {
	s, ok := g.secrets[key]
	if !ok {
		return apierrors.NewNotFound(schema.GroupResource{Resource: "secrets"}, key.Name)
	}
	*obj.(*corev1.Secret) = *s.DeepCopy()
	return nil
}

func registryWithSecretRef(ns string, ref xpv1.SecretKeySelector) *v1beta1.Registry {
	return &v1beta1.Registry{
		ObjectMeta: metav1.ObjectMeta{Name: "upstream", Namespace: ns},
		Spec: v1beta1.RegistrySpec{
			ForProvider: v1beta1.RegistryParameters{
				Name: "upstream",
				Type: "docker-registry",
				URL:  "https://registry.example.com",
				Credential: &v1beta1.RegistryCredential{
					AccessSecretRef: &ref,
				},
			},
		},
	}
}

func TestGetSecretFromRef(t *testing.T) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "upstream-creds", Namespace: "registries"},
		Data:       map[string][]byte{"token": []byte("s3cret-token")},
	}

	tests := []struct {
		name    string
		cr      *v1beta1.Registry
		wantVal string
		wantErr string
	}{
		{
			name: "reads the referenced key",
			cr: registryWithSecretRef("registries", xpv1.SecretKeySelector{
				SecretReference: xpv1.SecretReference{Name: "upstream-creds", Namespace: "registries"},
				Key:             "token",
			}),
			wantVal: "s3cret-token",
		},
		{
			name: "defaults the namespace to the registry's namespace",
			cr: registryWithSecretRef("registries", xpv1.SecretKeySelector{
				SecretReference: xpv1.SecretReference{Name: "upstream-creds"},
				Key:             "token",
			}),
			wantVal: "s3cret-token",
		},
		{
			name: "missing key is an error, not a placeholder",
			cr: registryWithSecretRef("registries", xpv1.SecretKeySelector{
				SecretReference: xpv1.SecretReference{Name: "upstream-creds", Namespace: "registries"},
				Key:             "password",
			}),
			wantErr: `key "password" not found`,
		},
		{
			name: "missing secret is an error",
			cr: registryWithSecretRef("registries", xpv1.SecretKeySelector{
				SecretReference: xpv1.SecretReference{Name: "absent", Namespace: "registries"},
				Key:             "token",
			}),
			wantErr: "secret registries/absent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ext := &external{kube: &secretGetter{secrets: map[client.ObjectKey]*corev1.Secret{
				{Namespace: "registries", Name: "upstream-creds"}: secret,
			}}}

			got, err := ext.getSecretFromRef(context.Background(), tt.cr)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("getSecretFromRef() error = %v, want it to contain %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("getSecretFromRef() error = %v", err)
			}
			if got != tt.wantVal {
				t.Errorf("getSecretFromRef() = %q, want %q", got, tt.wantVal)
			}
		})
	}
}
