/*
Copyright 2016 The Kubernetes Authors.

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

// Note: the example only works with the code within the same release/branch.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"

	v1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func main() {

	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	// 1 . check if exists
	// 2 . if not create
	// 3 . update

	// need: name of np, dns to check, pod selector

	for {

		addrs, err := net.LookupHost("chris.work")

		if err != nil {
			panic(err)
		}

		var to []v1.NetworkPolicyPeer
		for _, addr := range addrs {
			to = append(to, v1.NetworkPolicyPeer{IPBlock: &v1.IPBlock{CIDR: addr + "/32"}})
		}

		var networkPolicy *v1.NetworkPolicy

		networkPolicy, err = clientset.NetworkingV1().NetworkPolicies("default").Get(context.TODO(), "netdns-policy-generated", metav1.GetOptions{})

		if errors.IsNotFound(err) {
			networkPolicy, err = clientset.NetworkingV1().NetworkPolicies("default").Create(context.TODO(), &v1.NetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{Name: "netdns-policy-generated"}}, metav1.CreateOptions{})

		} else if err != nil {
			panic(err.Error())
		}

		networkPolicy.Spec = v1.NetworkPolicySpec{
			Egress: []v1.NetworkPolicyEgressRule{
				{To: to}},
			PodSelector: metav1.LabelSelector{MatchLabels: map[string]string{"role": "mysql-client"}}}

		s, _ := json.Marshal(networkPolicy)
		fmt.Println(string(s))

		clientset.NetworkingV1().NetworkPolicies("default").Update(context.TODO(), networkPolicy, metav1.UpdateOptions{})

		fmt.Println(err)
	}
}
