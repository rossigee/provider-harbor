/*
Copyright 2026 Crossplane Harbor Provider.
*/

package project

import (
	"context"

	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	projectv1beta1 "github.com/rossigee/provider-harbor/apis/project/v1beta1"
	registryv1beta1 "github.com/rossigee/provider-harbor/apis/registry/v1beta1"
)

const (
	errRegistryRefGet    = "cannot get Registry referenced by registryRef %q"
	errRegistryRefNoID   = "Registry %q referenced by registryRef has no Harbor ID yet"
	errRegistryRefNoKube = "cannot resolve registryRef without a Kubernetes client"
)

// resolveRegistryID returns the Harbor registry ID to use for a proxy cache
// project. An explicit registryId wins. Otherwise registryRef names a Registry
// in the project's namespace and its observed ID is used. A Registry that has
// not been created in Harbor yet is an error, so the reconcile is retried.
func (c *external) resolveRegistryID(ctx context.Context, cr *projectv1beta1.Project) (*int64, error) {
	if cr.Spec.ForProvider.RegistryID != nil {
		return cr.Spec.ForProvider.RegistryID, nil
	}
	if cr.Spec.ForProvider.RegistryRef == nil {
		return nil, nil
	}

	name := *cr.Spec.ForProvider.RegistryRef
	if c.kube == nil {
		return nil, errors.New(errRegistryRefNoKube)
	}

	reg := &registryv1beta1.Registry{}
	if err := c.kube.Get(ctx, client.ObjectKey{Namespace: cr.GetNamespace(), Name: name}, reg); err != nil {
		return nil, errors.Wrapf(err, errRegistryRefGet, name)
	}

	if reg.Status.AtProvider.ID == nil {
		return nil, errors.Errorf(errRegistryRefNoID, name)
	}

	id := *reg.Status.AtProvider.ID
	return &id, nil
}
