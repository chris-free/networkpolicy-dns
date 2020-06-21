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

	v1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func main() {
	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	pods, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("There are %d pods in the cluster\n", len(pods.Items))

	// Examples for error handling:
	// - Use helper functions e.g. errors.IsNotFound()
	// - And/or cast to StatusError and use its properties like e.g. ErrStatus.Message
	_, err = clientset.CoreV1().Pods("default").Get(context.TODO(), "example-xxxxx", metav1.GetOptions{})
	if errors.IsNotFound(err) {
		fmt.Printf("Pod example-xxxxx not found in default namespace\n")
	} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
		fmt.Printf("Error getting pod %v\n", statusError.ErrStatus.Message)
	} else if err != nil {
		panic(err.Error())
	} else {
		fmt.Printf("Found example-xxxxx pod in default namespace\n")
	}

	// 1 . check if exists
	// 2 . if not update
	// 3 .

	_, err2 := clientset.NetworkingV1().NetworkPolicies("default").Get(context.TODO(), "networkpolicy-generated", metav1.GetOptions{})

	if errors.IsNotFound(err2) {
		//panic(err2.Error())
	}

	newNetworkPolicy := v1.NetworkPolicy{Spec: v1.NetworkPolicySpec{
		Egress:      []v1.NetworkPolicyEgressRule{v1.NetworkPolicyEgressRule{To: []v1.NetworkPolicyPeer{v1.NetworkPolicyPeer{IPBlock: &v1.IPBlock{CIDR: "8.8.8.8/24"}}}}},
		PodSelector: metav1.LabelSelector{MatchLabels: map[string]string{"role": "mysql-client"}}}}

	s, _ := json.Marshal(newNetworkPolicy)
	fmt.Println(string(s))

	_, err3 := clientset.NetworkingV1().NetworkPolicies("default").Create(context.TODO(), &newNetworkPolicy, metav1.CreateOptions{})

	fmt.Println(err3)
}
