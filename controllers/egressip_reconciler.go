/*
Copyright 2021.

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

package controllers

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	egressipv1alpha1 "github.com/yingeli/egress-ip-operator/api/v1alpha1"
)

const (
	finalizer = "egressip.yingeli.github.com/finalizer"
)

func (r *EgressIPReconciler) reconcile(ctx context.Context, req ctrl.Request) error {
	var eip egressipv1alpha1.EgressIP
	if err := r.Get(ctx, req.NamespacedName, &eip); err != nil {
		return client.IgnoreNotFound(err)
	}

	deleted, err := r.checkDeletion(ctx, &eip)
	if err != nil {
		return err
	}
	if deleted {
		return nil
	}

	return r.createOrUpdate(ctx, &eip)
}

func (r *EgressIPReconciler) checkDeletion(ctx context.Context, eip *egressipv1alpha1.EgressIP) (bool, error) {
	if eip.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// registering our finalizer.
		if !containsString(eip.GetFinalizers(), finalizer) {
			controllerutil.AddFinalizer(eip, finalizer)
			if err := r.Update(ctx, eip); err != nil {
				return false, err
			}
		}
		return false, nil
	} else {
		// The object is being deleted
		if containsString(eip.GetFinalizers(), finalizer) {
			// our finalizer is present, so lets handle any external dependency
			if err := r.delete(ctx, eip); err != nil {
				// if fail to delete the external dependency here, return with error
				// so that it can be retried
				return true, err
			}

			// remove our finalizer from the list and update it.
			controllerutil.RemoveFinalizer(eip, finalizer)
			if err := r.Update(ctx, eip); err != nil {
				return true, err
			}
		}

		// Stop reconciliation as the item is being deleted
		return true, nil
	}
}

func (r *EgressIPReconciler) createOrUpdate(ctx context.Context, eip *egressipv1alpha1.EgressIP) error {
	var deployment appsv1.Deployment
	err := r.Get(ctx, getNamespacedName(eip), &deployment)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			return err
		}
		new_deployment := newEgressIPDeployment(eip)
		err := r.Create(ctx, new_deployment)
		if err != nil {
			return err
		}
	} else {
		updateEgressIPDeployment(&deployment, eip)
		err := r.Update(ctx, &deployment)
		if err != nil {
			return err
		}
	}

	var svc corev1.Service
	err = r.Get(ctx, getNamespacedName(eip), &svc)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			return err
		}
		new_svc := newEgressIPService(eip)
		err := r.Create(ctx, new_svc)
		if err != nil {
			return err
		}
	} else {
		updateEgressIPService(&svc, eip)
		err := r.Update(ctx, &svc)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *EgressIPReconciler) delete(ctx context.Context, eip *egressipv1alpha1.EgressIP) error {
	service := newEgressIPService(eip)
	err := r.Delete(ctx, service)
	client.IgnoreNotFound(err)
	if client.IgnoreNotFound(err) != nil {
		return err
	}

	deployment := newEgressIPDeployment(eip)
	return client.IgnoreNotFound(r.Delete(ctx, deployment))
}

func newEgressIPDeployment(eip *egressipv1alpha1.EgressIP) *appsv1.Deployment {
	deployment := appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      eip.Name,
			Namespace: eip.Namespace,
		},
	}
	updateEgressIPDeployment(&deployment, eip)
	return &deployment
}

func updateEgressIPDeployment(deployment *appsv1.Deployment, eip *egressipv1alpha1.EgressIP) {
	privileged := true
	seccurityContext := corev1.SecurityContext{
		Privileged: &privileged,
	}

	deployment.Spec = appsv1.DeploymentSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"app": getAppName(eip),
			},
		},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"app":            getAppName(eip),
					"egress-ip":      eip.Spec.IP,
					"egress-ip-name": eip.Name,
					"control-plane":  "controller-manager",
				},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{
					Image:           "yingeli/egress-ip-gateway",
					ImagePullPolicy: "Always",
					Name:            "gateway",
					Env:             getEnv(eip),
					Ports: []corev1.ContainerPort{
						{
							ContainerPort: 1701,
							Protocol:      "UDP",
							Name:          "l2tp",
						},
					},
					SecurityContext: &seccurityContext,
				}},
				InitContainers: []corev1.Container{{
					Image:           "yingeli/egress-ip-gateway",
					ImagePullPolicy: "Always",
					Name:            "gateway-init",
					Command: []string{
						"/init.sh",
					},
					Env: getEnv(eip),
				}},
				ServiceAccountName: "egress-ip-controller-manager",
			},
		},
	}
}

func getEnv(eip *egressipv1alpha1.EgressIP) []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name:  "EGRESS_IP_NAMESPACE",
			Value: eip.Namespace,
		},
		{
			Name:  "EGRESS_IP_NAME",
			Value: eip.Name,
		},
		{
			Name:  "EGRESS_PUBLIC_IP",
			Value: eip.Spec.IP,
		},
	}
}

func getAppName(eip *egressipv1alpha1.EgressIP) string {
	return eip.Name + "-gateway"
}

func newEgressIPService(eip *egressipv1alpha1.EgressIP) *corev1.Service {
	service := corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      eip.Name,
			Namespace: eip.Namespace,
		},
	}
	updateEgressIPService(&service, eip)
	return &service
}

func updateEgressIPService(service *corev1.Service, eip *egressipv1alpha1.EgressIP) {
	service.Spec = corev1.ServiceSpec{
		Selector: map[string]string{
			"app": getAppName(eip),
		},
		ClusterIP: "None",
	}
}

func getNamespacedName(eip *egressipv1alpha1.EgressIP) types.NamespacedName {
	return types.NamespacedName{
		Name:      eip.Name,
		Namespace: eip.Namespace,
	}
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

/*
func removeString(slice []string, s string) (result []string) {
    for _, item := range slice {
        if item == s {
            continue
        }
        result = append(result, item)
    }
    return
}
*/
