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

package apps

import (
	"context"

	"github.com/go-logr/logr"
	appsapplicationv1 "github.com/zhengyansheng/application-operators/api/apps/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// ApplicationReconciler reconciles a Application object
type ApplicationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	logger logr.Logger
}

// +kubebuilder:rbac:groups=apps.vk.io--owner,resources=applications,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps.vk.io--owner,resources=applications/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps.vk.io--owner,resources=applications/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Application object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.18.4/pkg/reconcile
func (r *ApplicationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	//r.logger = log.FromContext(ctx)
	klog.Info("Start Reconcile")

	// TODO(user): your logic here
	app := &appsapplicationv1.Application{}
	if err := r.Get(ctx, req.NamespacedName, app); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	{
		// Create Deployment
		deployment, err := r.genDeployment(app)
		if err != nil {
			return ctrl.Result{}, err
		}

		op, err := ctrl.CreateOrUpdate(ctx, r.Client, deployment, func() error {
			gen, err := r.genDeployment(app)
			if err != nil {
				klog.Warningf("generator created or updated err: %v", err)
				return err
			}
			deployment.Spec = gen.Spec
			return ctrl.SetControllerReference(app, deployment, r.Scheme) // 设置引用关系
		})
		if err != nil {
			return ctrl.Result{}, err
		}

		if op != controllerutil.OperationResultNone {
			klog.Infof("Resource created or updated, operation: %v", op)
		}
	}

	klog.Info("End Reconcile")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ApplicationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	//return ctrl.NewControllerManagedBy(mgr).
	//	For(&appsapplicationv1.Application{}). // 设置主资源
	//	Owns(&appsv1.Deployment{}).            // 设置从资源，可以支持多个
	//	Complete(r)
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsapplicationv1.Application{}).
		Complete(r)
}

func (r *ApplicationReconciler) genDeployment(app *appsapplicationv1.Application) (*appsv1.Deployment, error) {
	maxUnavailable := intstr.FromInt32(0)
	maxSurge := intstr.FromInt32(1)
	labels := map[string]string{"app": app.Name}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name,
			Namespace: app.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxUnavailable: &maxUnavailable,
					MaxSurge:       &maxSurge,
				},
			},
			Replicas: app.Spec.Replicas,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  app.Name,
							Image: app.Spec.Image,
						},
					},
				},
			},
		},
	}
	return deployment, nil
}
