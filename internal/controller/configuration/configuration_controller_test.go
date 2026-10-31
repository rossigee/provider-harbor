/*
Copyright 2026 Crossplane Harbor Provider.
*/

package configuration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	sdkmodels "github.com/goharbor/go-client/pkg/sdk/v2.0/models"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/rossigee/provider-harbor/apis/configuration/v1beta1"
	harborclients "github.com/rossigee/provider-harbor/internal/clients"
)

func strPtr(s string) *string { return &s }
func boolPtr(b bool) *bool    { return &b }

func strItem(v string) *sdkmodels.StringConfigItem {
	return &sdkmodels.StringConfigItem{Value: v}
}

func boolItem(v bool) *sdkmodels.BoolConfigItem {
	return &sdkmodels.BoolConfigItem{Value: v}
}

// currentOIDC is the Harbor configuration that matches testParams.
func currentOIDC() *sdkmodels.ConfigurationsResponse {
	return &sdkmodels.ConfigurationsResponse{
		AuthMode:        strItem("oidc_auth"),
		OIDCName:        strItem("Keycloak"),
		OIDCEndpoint:    strItem("https://sso.example.com/auth/realms/Example"),
		OIDCClientID:    strItem("harbor"),
		OIDCScope:       strItem("openid profile email"),
		OIDCUserClaim:   strItem("preferred_username"),
		OIDCGroupsClaim: strItem("groups"),
		OIDCAdminGroup:  strItem("harbor-admins"),
		OIDCGroupFilter: strItem(""),
		OIDCVerifyCert:  boolItem(true),
		OIDCAutoOnboard: boolItem(false),
		OIDCLogout:      boolItem(false),
	}
}

func testParams() v1beta1.ConfigurationParameters {
	return v1beta1.ConfigurationParameters{
		AuthMode: strPtr("oidc_auth"),
		OIDC: &v1beta1.OIDCSettings{
			Name:        strPtr("Keycloak"),
			Endpoint:    strPtr("https://sso.example.com/auth/realms/Example"),
			ClientID:    strPtr("harbor"),
			Scope:       strPtr("openid profile email"),
			UserClaim:   strPtr("preferred_username"),
			GroupsClaim: strPtr("groups"),
			AdminGroup:  strPtr("harbor-admins"),
			GroupFilter: strPtr(""),
			VerifyCert:  boolPtr(true),
			AutoOnboard: boolPtr(false),
			Logout:      boolPtr(false),
		},
	}
}

func hashOf(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func TestIsUpToDate(t *testing.T) {
	tests := []struct {
		name       string
		params     func() v1beta1.ConfigurationParameters
		current    func() *sdkmodels.ConfigurationsResponse
		secretHash string
		stored     string
		want       bool
	}{
		{
			name:    "everything matches",
			params:  testParams,
			current: currentOIDC,
			want:    true,
		},
		{
			name:   "auth mode differs",
			params: testParams,
			current: func() *sdkmodels.ConfigurationsResponse {
				c := currentOIDC()
				c.AuthMode = strItem("db_auth")
				return c
			},
			want: false,
		},
		{
			name:   "endpoint differs",
			params: testParams,
			current: func() *sdkmodels.ConfigurationsResponse {
				c := currentOIDC()
				c.OIDCEndpoint = strItem("https://other.example.com/realms/x")
				return c
			},
			want: false,
		},
		{
			name: "admin group cleared in spec is detected",
			params: func() v1beta1.ConfigurationParameters {
				p := testParams()
				p.OIDC.AdminGroup = strPtr("")
				return p
			},
			current: currentOIDC,
			want:    false,
		},
		{
			name: "unset spec fields are not compared",
			params: func() v1beta1.ConfigurationParameters {
				return v1beta1.ConfigurationParameters{AuthMode: strPtr("oidc_auth")}
			},
			current: func() *sdkmodels.ConfigurationsResponse {
				c := currentOIDC()
				c.OIDCEndpoint = strItem("anything")
				return c
			},
			want: true,
		},
		{
			name:   "verifyCert differs",
			params: testParams,
			current: func() *sdkmodels.ConfigurationsResponse {
				c := currentOIDC()
				c.OIDCVerifyCert = boolItem(false)
				return c
			},
			want: false,
		},
		{
			name: "client secret rotated since last write",
			params: func() v1beta1.ConfigurationParameters {
				p := testParams()
				p.OIDC.ClientSecretRef = &xpv1.SecretKeySelector{
					SecretReference: xpv1.SecretReference{Name: "oidc", Namespace: "crossplane-system"},
					Key:             "clientSecret",
				}
				return p
			},
			current:    currentOIDC,
			secretHash: hashOf("new"),
			stored:     hashOf("old"),
			want:       false,
		},
		{
			name: "client secret unchanged since last write",
			params: func() v1beta1.ConfigurationParameters {
				p := testParams()
				p.OIDC.ClientSecretRef = &xpv1.SecretKeySelector{
					SecretReference: xpv1.SecretReference{Name: "oidc", Namespace: "crossplane-system"},
					Key:             "clientSecret",
				}
				return p
			},
			current:    currentOIDC,
			secretHash: hashOf("same"),
			stored:     hashOf("same"),
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isUpToDate(tt.params(), tt.current(), tt.secretHash, tt.stored)
			if got != tt.want {
				t.Errorf("isUpToDate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildConfigurations(t *testing.T) {
	t.Run("unset fields are omitted", func(t *testing.T) {
		cfg := buildConfigurations(v1beta1.ConfigurationParameters{AuthMode: strPtr("oidc_auth")}, nil)
		if cfg.AuthMode == nil || *cfg.AuthMode != "oidc_auth" {
			t.Fatalf("AuthMode = %v, want oidc_auth", cfg.AuthMode)
		}
		if cfg.OIDCEndpoint != nil || cfg.OIDCClientSecret != nil || cfg.OIDCVerifyCert != nil {
			t.Errorf("unset OIDC fields must be nil so Harbor keeps them: %+v", cfg)
		}
	})

	t.Run("explicit empty string is sent to clear a value", func(t *testing.T) {
		p := testParams()
		p.OIDC.AdminGroup = strPtr("")
		cfg := buildConfigurations(p, nil)
		if cfg.OIDCAdminGroup == nil || *cfg.OIDCAdminGroup != "" {
			t.Errorf("OIDCAdminGroup = %v, want pointer to empty string", cfg.OIDCAdminGroup)
		}
	})

	t.Run("client secret is included when provided", func(t *testing.T) {
		secret := "s3cret"
		cfg := buildConfigurations(testParams(), &secret)
		if cfg.OIDCClientSecret == nil || *cfg.OIDCClientSecret != secret {
			t.Errorf("OIDCClientSecret = %v, want %q", cfg.OIDCClientSecret, secret)
		}
	})
}

// fakeService is a HarborClienter that records configuration writes. Only the
// configuration methods are exercised here.
func fakeService(current *sdkmodels.ConfigurationsResponse, getErr error, written **sdkmodels.Configurations, updateErr error) harborclients.HarborClienter {
	return &harborclients.MockHarborClient{
		GetConfigurationsFunc: func(ctx context.Context) (*sdkmodels.ConfigurationsResponse, error) {
			return current, getErr
		},
		UpdateConfigurationsFunc: func(ctx context.Context, cfg *sdkmodels.Configurations) error {
			if written != nil {
				*written = cfg
			}
			return updateErr
		},
	}
}

// secretKube is a client.Client that serves a single Secret from Get. Other
// methods panic if called, which keeps the tests honest about what the
// controller reads.
type secretKube struct {
	client.Client
	secret *corev1.Secret
}

func newSecretKube(secret *corev1.Secret) client.Client {
	return &secretKube{secret: secret}
}

func (k *secretKube) Get(_ context.Context, key client.ObjectKey, obj client.Object, _ ...client.GetOption) error {
	if k.secret == nil || key.Name != k.secret.Name || key.Namespace != k.secret.Namespace {
		return apierrors.NewNotFound(schema.GroupResource{Resource: "secrets"}, key.Name)
	}
	s, ok := obj.(*corev1.Secret)
	if !ok {
		panic("secretKube only serves Secret objects")
	}
	*s = *k.secret.DeepCopy()
	return nil
}

func oidcSecret(value string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "oidc", Namespace: "crossplane-system"},
		Data:       map[string][]byte{"clientSecret": []byte(value)},
	}
}

func configWithSecret() *v1beta1.Configuration {
	p := testParams()
	p.OIDC.ClientSecretRef = &xpv1.SecretKeySelector{
		SecretReference: xpv1.SecretReference{Name: "oidc", Namespace: "crossplane-system"},
		Key:             "clientSecret",
	}
	return &v1beta1.Configuration{
		ObjectMeta: metav1.ObjectMeta{Name: "harbor", Namespace: "crossplane-system"},
		Spec:       v1beta1.ConfigurationSpec{ForProvider: p},
	}
}

func TestObserve(t *testing.T) {
	t.Run("reads auth mode and reports up to date after a recorded write", func(t *testing.T) {
		cr := configWithSecret()
		stored := hashOf("s3cret")
		cr.Status.AtProvider.OIDCClientSecretHash = &stored

		ext := &external{
			service: fakeService(currentOIDC(), nil, nil, nil),
			kube:    newSecretKube(oidcSecret("s3cret")),
		}

		obs, err := ext.Observe(context.Background(), cr)
		if err != nil {
			t.Fatalf("Observe() error = %v", err)
		}
		if !obs.ResourceExists || !obs.ResourceUpToDate {
			t.Errorf("Observe() = %+v, want exists and up to date", obs)
		}
		if cr.Status.AtProvider.AuthMode == nil || *cr.Status.AtProvider.AuthMode != "oidc_auth" {
			t.Errorf("AuthMode in status = %v, want oidc_auth", cr.Status.AtProvider.AuthMode)
		}
	})

	t.Run("detects rotated client secret", func(t *testing.T) {
		cr := configWithSecret()
		stored := hashOf("old-secret")
		cr.Status.AtProvider.OIDCClientSecretHash = &stored

		ext := &external{
			service: fakeService(currentOIDC(), nil, nil, nil),
			kube:    newSecretKube(oidcSecret("new-secret")),
		}

		obs, err := ext.Observe(context.Background(), cr)
		if err != nil {
			t.Fatalf("Observe() error = %v", err)
		}
		if obs.ResourceUpToDate {
			t.Errorf("Observe() up to date after secret rotation, want not up to date")
		}
	})

	t.Run("surfaces Harbor read errors", func(t *testing.T) {
		ext := &external{
			service: fakeService(nil, errors.New("connection refused"), nil, nil),
			kube:    newSecretKube(nil),
		}

		_, err := ext.Observe(context.Background(), configWithSecret())
		if err == nil || !strings.Contains(err.Error(), errGetConfig) {
			t.Errorf("Observe() error = %v, want it to wrap %q", err, errGetConfig)
		}
	})

	t.Run("fails when the referenced secret is missing", func(t *testing.T) {
		ext := &external{
			service: fakeService(currentOIDC(), nil, nil, nil),
			kube:    newSecretKube(nil),
		}

		_, err := ext.Observe(context.Background(), configWithSecret())
		if err == nil || !strings.Contains(err.Error(), errGetSecret) {
			t.Errorf("Observe() error = %v, want it to wrap %q", err, errGetSecret)
		}
	})
}

func TestCreateUpdateRecordsSecretHash(t *testing.T) {
	var written *sdkmodels.Configurations
	cr := configWithSecret()

	ext := &external{
		service: fakeService(currentOIDC(), nil, &written, nil),
		kube:    newSecretKube(oidcSecret("s3cret")),
	}

	if _, err := ext.Update(context.Background(), cr); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if written == nil || written.OIDCClientSecret == nil || *written.OIDCClientSecret != "s3cret" {
		t.Fatalf("Update() did not send the client secret to Harbor: %+v", written)
	}
	if cr.Status.AtProvider.OIDCClientSecretHash == nil || *cr.Status.AtProvider.OIDCClientSecretHash != hashOf("s3cret") {
		t.Errorf("secret hash not recorded after write: %v", cr.Status.AtProvider.OIDCClientSecretHash)
	}
}

func TestCreatePropagatesHarborError(t *testing.T) {
	cr := &v1beta1.Configuration{
		ObjectMeta: metav1.ObjectMeta{Name: "harbor"},
		Spec:       v1beta1.ConfigurationSpec{ForProvider: testParams()},
	}

	ext := &external{
		service: fakeService(nil, nil, nil, errors.New("forbidden")),
		kube:    newSecretKube(nil),
	}

	_, err := ext.Create(context.Background(), cr)
	if err == nil || !strings.Contains(err.Error(), errUpdateConfig) {
		t.Errorf("Create() error = %v, want it to wrap %q", err, errUpdateConfig)
	}
}

func TestDeleteLeavesHarborUntouched(t *testing.T) {
	var written *sdkmodels.Configurations
	ext := &external{
		service: fakeService(currentOIDC(), nil, &written, nil),
		kube:    newSecretKube(nil),
	}

	if _, err := ext.Delete(context.Background(), configWithSecret()); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if written != nil {
		t.Errorf("Delete() wrote configuration to Harbor: %+v", written)
	}
}
