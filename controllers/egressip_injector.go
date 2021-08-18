/*
Copyright 2018 The Kubernetes Authors.

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
	"encoding/json"

	//"fmt"
	"net/http"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	egressipv1alpha1 "github.com/yingeli/egress-ip-operator/api/v1alpha1"
)

//+kubebuilder:webhook:path=/mutate-v1-pod,mutating=true,failurePolicy=fail,groups="",resources=pods,verbs=create;update,versions=v1,name=mpod.kb.io,sideEffects=none,admissionReviewVersions=v1
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete

// podAnnotator annotates Pods
type EgressIPInjector struct {
	Client  client.Client
	decoder *admission.Decoder
}

// PodAnnotator adds an annotation to every incoming pods.
func (a *EgressIPInjector) Handle(ctx context.Context, req admission.Request) admission.Response {
	pod := &corev1.Pod{}

	err := a.decoder.Decode(req, pod)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	eip, err := a.selectEgressIP(ctx, pod)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}
	if eip == nil {
		return admission.Allowed("")
	}

	privileged := true
	director := corev1.Container{
		Name:            "egress-ip-director",
		Image:           "yingeli/egress-ip-director",
		ImagePullPolicy: "Always",
		SecurityContext: &corev1.SecurityContext{
			Privileged: &privileged,
		},
		Env: []corev1.EnvVar{
			{
				Name:  "EGRESS_GATEWAY",
				Value: eip.Name + "." + eip.Namespace,
			},
			{
				Name:  "LOCAL_NETWORK",
				Value: "10.0.0.0/8",
			},
		},
	}
	pod.Spec.Containers = append(pod.Spec.Containers, director)

	if pod.Spec.Affinity == nil {
		pod.Spec.Affinity = &corev1.Affinity{}
	}

	if pod.Spec.Affinity.PodAffinity == nil {
		pod.Spec.Affinity.PodAffinity = &corev1.PodAffinity{}
	}

	term := corev1.WeightedPodAffinityTerm{
		Weight: 100,
		PodAffinityTerm: corev1.PodAffinityTerm{
			LabelSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"egress-ip-name": eip.Name,
				},
			},
			Namespaces:  []string{eip.Namespace},
			TopologyKey: "kubernetes.io/hostname",
		},
	}

	pod.Spec.Affinity.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution = append(
		pod.Spec.Affinity.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution,
		term)

	marshaledPod, err := json.Marshal(pod)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}

// EgressIPInjector implements admission.DecoderInjector.
// A decoder will be automatically injected.

// InjectDecoder injects the decoder.
func (a *EgressIPInjector) InjectDecoder(d *admission.Decoder) error {
	a.decoder = d
	return nil
}

func (a *EgressIPInjector) selectEgressIP(ctx context.Context, pod *corev1.Pod) (eip *egressipv1alpha1.EgressIP, err error) {
	var eips egressipv1alpha1.EgressIPList
	err = a.Client.List(ctx, &eips)
	if err != nil {
		return eip, err
	}

	for _, ip := range eips.Items {
		selector := labels.SelectorFromSet(ip.Spec.PodSelector.MatchLabels)
		lables := labels.Set(pod.Labels)
		if selector.Matches(lables) {
			return &ip, nil
		}
	}
	return nil, nil
}
