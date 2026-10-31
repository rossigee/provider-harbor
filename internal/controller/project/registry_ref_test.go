/*
Copyright 2026 Crossplane Harbor Provider.
*/

package project

import (
	"context"
	"strings"
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	projectv1beta1 "github.com/rossigee/provider-harbor/apis/project/v1beta1"
	registryv1beta1 "github.com/rossigee/provider-harbor/apis/registry/v1beta1"
)

// registryGetter serves Registry objects from Get. Other methods are unused by
// resolveRegistryID and panic if reached.
type registryGetter struct {
	client.Client
	registries map[client.ObjectKey]*registryv1beta1.Registry
}

func (g *registryGetter) Get(_ context.Context, key client.ObjectKey, obj client.Object, _ ...client.GetOption) error {
	reg, ok := g.registries[key]
	if !ok {
		return apierrors.NewNotFound(schema.GroupResource{Group: "registry.harbor.m.crossplane.io", Resource: "registries"}, key.Name)
	}
	*obj.(*registryv1beta1.Registry) = *reg.DeepCopy()
	return nil
}

func int64Ptr(i int64) *int64  { return &i }
func strPtrP(s string) *string { return &s }

func projectReferencing(ns string, registryID *int64, registryRef *string) *projectv1beta1.Project {
	p := &projectv1beta1.Project{ObjectMeta: metav1.ObjectMeta{Name: "proxy", Namespace: ns}}
	p.Spec.ForProvider.Name = "proxy"
	p.Spec.ForProvider.RegistryID = registryID
	p.Spec.ForProvider.RegistryRef = registryRef
	return p
}

func TestResolveRegistryID(t *testing.T) {
	withID := &registryv1beta1.Registry{
		ObjectMeta: metav1.ObjectMeta{Name: "dockerhub", Namespace: "registries"},
		Status: registryv1beta1.RegistryStatus{
			AtProvider: registryv1beta1.RegistryObservation{ID: int64Ptr(7)},
		},
	}
	noID := &registryv1beta1.Registry{
		ObjectMeta: metav1.ObjectMeta{Name: "pending", Namespace: "registries"},
	}

	kube := &registryGetter{registries: map[client.ObjectKey]*registryv1beta1.Registry{
		{Namespace: "registries", Name: "dockerhub"}: withID,
		{Namespace: "registries", Name: "pending"}:   noID,
	}}

	tests := []struct {
		name    string
		cr      *projectv1beta1.Project
		kube    client.Client
		want    *int64
		wantErr string
	}{
		{
			name: "no registry configured",
			cr:   projectReferencing("registries", nil, nil),
			kube: kube,
			want: nil,
		},
		{
			name: "explicit registryId wins over registryRef",
			cr:   projectReferencing("registries", int64Ptr(3), strPtrP("dockerhub")),
			kube: kube,
			want: int64Ptr(3),
		},
		{
			name: "registryRef resolves to the Registry's observed ID",
			cr:   projectReferencing("registries", nil, strPtrP("dockerhub")),
			kube: kube,
			want: int64Ptr(7),
		},
		{
			name:    "registryRef to a Registry with no ID yet is an error",
			cr:      projectReferencing("registries", nil, strPtrP("pending")),
			kube:    kube,
			wantErr: "has no Harbor ID yet",
		},
		{
			name:    "registryRef to a missing Registry is an error",
			cr:      projectReferencing("registries", nil, strPtrP("absent")),
			kube:    kube,
			wantErr: "cannot get Registry referenced by registryRef",
		},
		{
			name:    "registryRef without a Kubernetes client is an error",
			cr:      projectReferencing("registries", nil, strPtrP("dockerhub")),
			kube:    nil,
			wantErr: "without a Kubernetes client",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ext := &external{kube: tt.kube}

			got, err := ext.resolveRegistryID(context.Background(), tt.cr)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("resolveRegistryID() error = %v, want it to contain %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveRegistryID() error = %v", err)
			}
			switch {
			case tt.want == nil && got != nil:
				t.Errorf("resolveRegistryID() = %d, want nil", *got)
			case tt.want != nil && (got == nil || *got != *tt.want):
				t.Errorf("resolveRegistryID() = %v, want %d", got, *tt.want)
			}
		})
	}
}
