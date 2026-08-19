/*
Copyright 2025 The Crossplane Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package providerconfig

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"

	"github.com/rossigee/provider-harbor/apis/v1beta1"
)

const controllerName = "providerconfig.harbor.crossplane.io"

// Setup registers the ProviderConfig controller.
func Setup(mgr ctrl.Manager) error {
	r := &reconciler{kube: mgr.GetClient(), logger: mgr.GetLogger()}
	return ctrl.NewControllerManagedBy(mgr).
		Named(controllerName).
		For(&v1beta1.ProviderConfig{}).
		Complete(r)
}

type reconciler struct {
	kube   client.Client
	logger logr.Logger
}

func (r *reconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	log := r.logger.WithValues("providerconfig", req.Name)
	log.Info("reconciling ProviderConfig")

	pc := &v1beta1.ProviderConfig{}
	if err := r.kube.Get(ctx, req.NamespacedName, pc); err != nil {
		log.Error(err, "failed to get ProviderConfig")
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}

	log.Info("ProviderConfig available")
	pc.Status.SetConditions(xpv1.Available())

	fresh := &v1beta1.ProviderConfig{}
	if err := r.kube.Get(ctx, client.ObjectKey{Name: pc.GetName()}, fresh); err != nil {
		log.Error(err, "failed to get latest ProviderConfig for status update")
		return reconcile.Result{RequeueAfter: 30 * time.Second}, client.IgnoreNotFound(err)
	}

	fresh.Status = pc.Status
	if err := r.kube.Status().Update(ctx, fresh); err != nil {
		log.Error(err, "failed to update ProviderConfig status")
		if errors.IsConflict(err) {
			log.Info("conflict updating, will retry")
			return reconcile.Result{RequeueAfter: 30 * time.Second}, nil
		}
		return reconcile.Result{RequeueAfter: 30 * time.Second}, nil
	}

	return reconcile.Result{}, nil
}
