package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"time"

	"gopkg.in/yaml.v2"
	v1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type PodSelector struct {
	MatchLabels map[string]string `yaml:"matchLabels"`
}

type Settings struct {
	Domain      []string    `yaml:"domain"`
	PodSelector PodSelector `yaml:"podSelector"`
}

// todo
// 1. check for configmap changes then reset loop
// 2. before updating do a diff to find changes
// 3. add timer variable
// 4. name

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

		settingsBytes, err := ioutil.ReadFile("/configmap/settings.yml")

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
				fmt.Println("Error looking up domain: " + domain + err.Error() )
				continue
			}

			for _, addr := range addr {
				addrs = append(addrs, addr)
			}
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
			PodSelector: settings.PodSelector}}


		// do a compare whether it needs to be updated
		
		_, err = clientset.NetworkingV1().NetworkPolicies("default").Update(context.TODO(), networkPolicy, metav1.UpdateOptions{})

		if err != nil {
			fmt.Println("Error updating policy:" + err.Error())
		}

		time.Sleep(1 * time.Minute)
	}
}
