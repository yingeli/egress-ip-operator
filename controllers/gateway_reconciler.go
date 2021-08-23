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
	"os"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/coreos/go-iptables/iptables"
	"github.com/go-logr/logr"
	"github.com/yingeli/egress-ip-operator/providers"
	"github.com/yingeli/egress-ip-operator/providers/azure"

	egressipclients "github.com/yingeli/egress-ip-operator/clients"
)

type GatewayReconciler struct {
	client       *client.Client
	eipc         egressipclients.EgressIPClient
	provider     providers.Provider
	log          logr.Logger
	nodeName     string
	localNetwork string
}

func GatewayPodSelectors() (fs fields.Selector, ls labels.Selector, err error) {
	ls, err = labels.Parse("egress-ip")
	if err != nil {
		return fs, ls, err
	}

	fs = fields.SelectorFromSet(fields.Set{"spec.nodeName": os.Getenv("NODE_NAME")})

	return fs, ls, nil
}

func openGatewayReconciler(client *client.Client, provider providers.Provider) (r GatewayReconciler, err error) {
	ctx := context.Background()
	eipc, err := egressipclients.OpenEgressIPClient(ctx)
	if err != nil {
		return r, err
	}
	r = GatewayReconciler{
		client:       client,
		eipc:         eipc,
		provider:     provider,
		log:          ctrl.Log.WithName("gateway-reconciler"),
		nodeName:     os.Getenv("NODE_NAME"),
		localNetwork: os.Getenv("LOCAL_NETWORK"),
	}
	return r, nil
}

func openAzureGatewayReconciler(client *client.Client) (GatewayReconciler, error) {
	provider := azure.NewProvider()
	return openGatewayReconciler(client, &provider)
}

func (r *GatewayReconciler) reconcile(ctx context.Context, req ctrl.Request) error {
	podMap, err := r.getPodMap(ctx, req.Namespace)
	if err != nil {
		return err
	}

	ipt, err := iptables.New()
	if err != nil {
		return err
	}
	ruleMap, err := r.getSNATRuleMap(ipt)
	if err != nil {
		return err
	}

	for podIP, rule := range ruleMap {
		if _, exist := podMap[podIP]; !exist {
			if err := r.dissociate(ctx, ipt, rule); err != nil {
				return err
			}
		}
	}

	for podIP, pod := range podMap {
		if _, exist := ruleMap[podIP]; !exist {
			//r.log.Info("entering associate", "pod", pod)
			if err := r.associate(ctx, ipt, &pod); err != nil {
				return err
			}
		}
	}

	return nil
}

func (r *GatewayReconciler) getPodMap(ctx context.Context, namespace string) (m map[string]corev1.Pod, err error) {
	//var pods corev1.PodList
	//if err := r.List(ctx, &pods, client.InNamespace(req.Namespace),
	//	client.MatchingFieldsSelector{Selector: fs}, client.MatchingLabelsSelector{Selector: ls}); err != nil {
	//	return err
	//}
	var pods corev1.PodList
	if err := (*r.client).List(ctx, &pods, client.InNamespace(namespace)); err != nil {
		return m, err
	}

	m = make(map[string]corev1.Pod)
	for _, pod := range pods.Items {
		egressip := pod.Labels["egress-ip"]
		if pod.Spec.NodeName != r.nodeName || egressip == "" {
			r.log.Info("skipping non-gateway pod", "nodeName", pod.Spec.NodeName, "egress-ip", egressip)
			continue
		}
		m[pod.Status.PodIP] = pod
		//r.log.Info("gateway pod added", "nodeName", pod.Spec.NodeName, "egress-ip", egressip)
	}
	return m, nil
}

func (r *GatewayReconciler) getSNATRuleMap(ipt *iptables.IPTables) (m map[string]SNATRule, err error) {
	rules, err := ipt.List("nat", "POSTROUTING")
	if err != nil {
		return m, err
	}

	m = make(map[string]SNATRule)
	for _, rule := range rules {
		if sr, ok := ParseSNATRule(rule); ok {
			m[sr.Source] = sr
		}
	}
	return m, nil
}

func (r *GatewayReconciler) associate(ctx context.Context, ipt *iptables.IPTables, pod *corev1.Pod) error {
	egressIP := pod.Labels["egress-ip"]
	podIP := pod.Status.PodIP
	if egressIP == "" || podIP == "" {
		return nil
	}

	srcIP, err := r.provider.Associate(ctx, egressIP, podIP)
	if err != nil {
		return err
	}

	rule := NewSNATRule(podIP, r.localNetwork, srcIP)
	if err := ipt.Insert("nat", "POSTROUTING", 1, rule.Spec()...); err != nil {
		return err
	}

	namespace := pod.Labels["egress-ip-namespace"]
	name := pod.Labels["egress-ip-name"]
	if err := r.updateEgressIPStatusPhase(ctx, namespace, name, "Configured"); err != nil {
		r.log.Error(err, "error updating EgressIP status phase", "egress-ip-namespace", namespace, "egress-ip", name)
		return err
	}

	r.log.Info("associated EgressIP successfuly", "EgressIP", egressIP)
	return nil
}

func (r *GatewayReconciler) dissociate(ctx context.Context, ipt *iptables.IPTables, rule SNATRule) error {
	if err := ipt.Delete("nat", "POSTROUTING", rule.Spec()...); err != nil {
		return err
	}
	if err := r.provider.Dissociate(ctx, rule.ToSource); err != nil {
		return err
	}
	r.log.Info("dissociated EgressIP successfuly", "Private IP", rule.ToSource)
	return nil
}

func (r *GatewayReconciler) updateEgressIPStatusPhase(ctx context.Context, namespace, name, phase string) error {
	eip, err := r.eipc.GetEgressIP(ctx, namespace, name)
	if err != nil {
		r.log.Error(err, "error updating EgressIP phase", "namespace", namespace, "name", name)
		return err
	}
	eip.Status.Phase = phase
	if err := eip.UpdateStatus(ctx); err != nil {
		return err
	}
	return nil
}
