/*
Copyright 2024.

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

package controller

import (
	"context"
	"reflect"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	theobserversworldv1 "github.com/the-observers/template-resource-poc/api/v1"
)

// PocThingReconciler reconciles a PocThing object
type PocThingReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=the-observers.world,resources=pocthings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=the-observers.world,resources=poctemplates,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=the-observers.world,resources=pocthings/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=the-observers.world,resources=pocthings/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the PocThing object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *PocThingReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Starting reconciliation", "Namespace", req.Namespace, "Name", req.Name)

	var thing theobserversworldv1.PocThing
	var templ theobserversworldv1.PocTemplate

	err := r.Get(ctx, req.NamespacedName, &thing)
	if err != nil {
		return ctrl.Result{}, err
	}

	log.Info("Found PocThing", "Namespace", thing.Namespace, "Name", thing.Name)

	err = r.Get(ctx, types.NamespacedName{Name: thing.Spec.Template, Namespace: thing.ObjectMeta.Namespace}, &templ)
	if err != nil {
		return ctrl.Result{}, err
	}

	log.Info("Found PocTemplate", "Namespace", templ.Namespace, "Name", templ.Name)

	deploymentName := thing.ObjectMeta.Name + "-deployment"
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentName,
			Namespace: thing.Namespace,
		},
		Spec: *templ.Spec.DeepCopy(),
	}

	dep.Spec.Template.ObjectMeta.Labels["demo"] = thing.Spec.Demo

	err = controllerutil.SetControllerReference(&thing, dep, r.Scheme)
	if err != nil {
		return ctrl.Result{}, err
	}

	var existing appsv1.Deployment
	err = r.Get(ctx, types.NamespacedName{
		Name:      deploymentName,
		Namespace: thing.Namespace,
	}, &existing)

	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			log.Info("Creating a new Deployment", "Namespace", dep.Namespace, "Name", dep.Name)
			if err := r.Create(ctx, dep); err != nil {
				return ctrl.Result{}, err
			}
		} else {
			return ctrl.Result{}, err
		}
	} else {
		if !reflect.DeepEqual(dep.Spec, existing.Spec) {
			log.Info("Updating a Deployment", "Namespace", dep.Namespace, "Name", dep.Name)
			existing.Spec = dep.Spec
			if err := r.Update(ctx, &existing); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PocThingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&theobserversworldv1.PocThing{}).
		Complete(r)
}
