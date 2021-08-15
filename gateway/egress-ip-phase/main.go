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

package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	//metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	egressipv1alpha1 "github.com/yingeli/egress-ip-operator/api/v1alpha1"
	egressipclients "github.com/yingeli/egress-ip-operator/clients"
)

const (
	usage = "usage: egress-ip-phase namespace name get; egress-ip-phase namespace name update NEW_PHASE; usage: egress-ip-phase namespace name wait DESIRED_PHASE"
)

var (
	scheme = runtime.NewScheme()
	log    = ctrl.Log.WithName("egress-ip-phase")
)

func init() {
	utilruntime.Must(egressipv1alpha1.AddToScheme(scheme))
}

func main() {
	if !(len(os.Args) >= 5 || len(os.Args) >= 4 && os.Args[3] == "get") {
		fmt.Printf("not enough args\n")
		log.Error(fmt.Errorf("not enough args"), usage)
		os.Exit(1)
	}

	namespace := os.Args[1]
	name := os.Args[2]
	action := os.Args[3]
	phase := ""
	if len(os.Args) >= 5 {
		phase = os.Args[4]
	}

	c, err := client.New(ctrl.GetConfigOrDie(), client.Options{Scheme: scheme})
	if err != nil {
		fmt.Printf("error new client: %v\n", err)
		log.Error(err, "error new client")
		os.Exit(1)
	}

	ctx := context.Background()

	eipc, err := egressipclients.OpenEgressIPClient(ctx, &c, namespace, name)
	if err != nil {
		fmt.Printf("error getting EgressIP resource: %v\n", err)
		log.Error(err, "error getting EgressIP resource")
		os.Exit(1)
	}

	switch action {
	case "get":
		fmt.Printf(eipc.Status.Phase)
	case "update":
		eipc.Status.Phase = phase
		err = eipc.UpdateStatus(ctx)
		if err != nil {
			fmt.Printf("error updating EgressIP phase: %v\n", err)
			log.Error(err, "error updating EgressIP phase")
			os.Exit(1)
		}
	case "wait":
		for i := 0; i < 600; i++ {
			if eipc.Status.Phase == phase {
				return
			}
			log.Info("eipc.Status.Phase not ready", "eipc.Status.Phase", eipc.Status.Phase)
			time.Sleep(time.Second)
			if err := eipc.Refresh(ctx); err != nil {
				fmt.Printf("error refreshing: %v\n", err)
				log.Error(err, "error refreshing")
				os.Exit(1)
			}
		}
		log.Error(fmt.Errorf("wait timeout"), "wait timeout")
		os.Exit(1)
	default:
		fmt.Printf("invalid action. " + usage + "\n")
		log.Error(fmt.Errorf("invalid action"), "invalid action")
		os.Exit(1)
	}
}
