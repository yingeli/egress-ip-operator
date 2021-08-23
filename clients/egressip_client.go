/*
Copyright 2021 Ying Ge Li.

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

package clients

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	egressipv1alpha1 "github.com/yingeli/egress-ip-operator/api/v1alpha1"
)

// EgressIPClient
type EgressIPClient struct {
	client.Client
}

type EgressIP struct {
	client *EgressIPClient
	egressipv1alpha1.EgressIP
	key types.NamespacedName
}

func OpenEgressIPClient(ctx context.Context) (eipc EgressIPClient, err error) {
	scheme := runtime.NewScheme()
	utilruntime.Must(egressipv1alpha1.AddToScheme(scheme))

	c, err := client.New(ctrl.GetConfigOrDie(), client.Options{Scheme: scheme})
	if err != nil {
		fmt.Printf("error creating client. error: %v\n", err)
		return eipc, err
	}
	eipc = EgressIPClient{c}
	return eipc, nil
}

func (c *EgressIPClient) GetEgressIP(ctx context.Context, namespace, name string) (eip EgressIP, err error) {
	eip = EgressIP{
		client:   c,
		EgressIP: egressipv1alpha1.EgressIP{},
		key: types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		},
	}
	if err := eip.Refresh(ctx); err != nil {
		fmt.Printf("error refreshing EgressIP. Namespace: %s, Name: %s, error: %v\n", eip.key.Namespace, eip.key.Name, err)
		return eip, err
	}
	return eip, nil
}

func (e *EgressIP) Refresh(ctx context.Context) error {
	return e.client.Get(ctx, e.key, &e.EgressIP)
}

func (e *EgressIP) UpdateStatus(ctx context.Context) error {
	return e.client.Status().Update(ctx, &e.EgressIP)
}
