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

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	egressipv1alpha1 "github.com/yingeli/egress-ip-operator/api/v1alpha1"
)

// EgressIPClient
type EgressIPClient struct {
	egressipv1alpha1.EgressIP
	client *client.Client
	key    types.NamespacedName
}

func OpenEgressIPClient(ctx context.Context, c *client.Client, namespace, name string) (eipc EgressIPClient, err error) {
	k := types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}
	eipc = EgressIPClient{
		EgressIP: egressipv1alpha1.EgressIP{},
		client:   c,
		key:      k,
	}
	if err := eipc.Refresh(ctx); err != nil {
		return eipc, err
	}
	return eipc, nil
}

func (c *EgressIPClient) Refresh(ctx context.Context) error {
	return (*c.client).Get(ctx, c.key, &c.EgressIP)
}

func (c *EgressIPClient) UpdateStatus(ctx context.Context) error {
	return (*c.client).Status().Update(ctx, &c.EgressIP)
}
