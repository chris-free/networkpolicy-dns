package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"reflect"
	"sort"
	"time"

	"github.com/ghodss/yaml"
	v1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type Settings struct {
	Domain      []string             `json:"domain"`
	PodSelector metav1.LabelSelector `json:"podSelector" protobuf:"bytes,1,opt,name=podSelector"`
}

// todo
// 1. check for configmap changes then reset loop
// 3. add timer variable
// 4. name
// 5. use: https://github.com/kubernetes/apimachinery/blob/master/pkg/util/yaml/decoder.go for unmarshalling settings

func main() {
	var err error

	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	for {
		var settings Settings

		//settingsBytes, err := ioutil.ReadFile("/configmap/settings.yml")

		settingsBytes, err := ioutil.ReadFile("/app/settings.yml")

		if err != nil {
			fmt.Println("Error opening settings: " + err.Error())
			continue
		}

		err = yaml.Unmarshal(settingsBytes, &settings)

		if err != nil {
			fmt.Println("Error unmarshalling settings: " + err.Error())
			continue
		}

		var addrs []string

		for _, domain := range settings.Domain {
			addr, err := net.LookupHost(domain)

			if err != nil {
				fmt.Println("Error looking up domain: " + domain + err.Error())
				continue
			}

			for _, addr := range addr {
				addrs = append(addrs, addr)
			}
		}

		sort.Strings(addrs)

		var to []v1.NetworkPolicyPeer
		for _, addr := range addrs {
			to = append(to, v1.NetworkPolicyPeer{IPBlock: &v1.IPBlock{CIDR: addr + "/32"}})
		}

		var networkPolicy *v1.NetworkPolicy

		var networkPolicy2 *v1.NetworkPolicy

		networkPolicy2, err = clientset.NetworkingV1().NetworkPolicies("default").Get(context.TODO(), "netdns-policy-generated", metav1.GetOptions{})

		networkPolicy, err = clientset.NetworkingV1().NetworkPolicies("default").Get(context.TODO(), "netdns-policy-generated", metav1.GetOptions{})
		if errors.IsNotFound(err) {
			networkPolicy, err = clientset.NetworkingV1().NetworkPolicies("default").Create(context.TODO(), &v1.NetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{Name: "netdns-policy-generated"}}, metav1.CreateOptions{})

			if err != nil {
				fmt.Println("Error creating policy: " + err.Error())
				continue
			}

		} else if err != nil {
			fmt.Println("Error getting policy: " + err.Error())
			continue
		}

		networkPolicy.Spec = v1.NetworkPolicySpec{
			Egress: []v1.NetworkPolicyEgressRule{
				{To: to}},
			PodSelector: settings.PodSelector}

		update := false

		if !reflect.DeepEqual(networkPolicy.Spec.Egress, networkPolicy2.Spec.Egress) {
			fmt.Println("LOG: DNS change, updating NetworkPolicy")
			update = true
		}

		if !reflect.DeepEqual(networkPolicy.Spec.PodSelector, networkPolicy2.Spec.PodSelector) {
			fmt.Println("LOG: PodSelector change, updating NetworkPolicy")
			update = true
		}

		if update {
			_, err = clientset.NetworkingV1().NetworkPolicies("default").Update(context.TODO(), networkPolicy, metav1.UpdateOptions{})

			if err != nil {
				fmt.Println("Error updating policy:" + err.Error())
			}
		}

		time.Sleep(1 * time.Minute)
	}
}
