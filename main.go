package main

import (
	"context"
	"io/ioutil"
	"log"
	"net"
	"reflect"
	"sort"
	"time"

	v1 "k8s.io/api/networking/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

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

	reset := make(chan bool)

	watchSettings(reset)

	settings, err := readSettings()

	if err != nil {
		return
	}

	var ticker *time.Ticker
	ticker = time.NewTicker(time.Duration(settings.interval) * time.Second)

	for {
		select {
		case <-reset:
			ticker.Stop()
			ticker = time.NewTicker(time.Duration(settings.interval) * time.Second)
		case <-ticker.C:
			log.Println("tick")
			run(clientset)
		}
	}
}

func run(clientset *kubernetes.Clientset) {

	settings, err := readSettings()

	if err != nil {
		log.Println(err.Error())
		return
	}

	var addrs []string

	for _, domain := range settings.Domain {
		addr, err := net.LookupHost(domain)

		if err != nil {
			log.Println("Error looking up domain: " + domain + err.Error())
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

	namespaceBytes, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")

	if err != nil {
		log.Println("Error reading namespace service account file")
		return
	}

	namespace := string(namespaceBytes)

	networkPolicy2, err = clientset.NetworkingV1().NetworkPolicies(namespace).Get(context.TODO(), "netdns-policy-generated", metav1.GetOptions{})

	networkPolicy, err = clientset.NetworkingV1().NetworkPolicies(namespace).Get(context.TODO(), "netdns-policy-generated", metav1.GetOptions{})
	if apiErrors.IsNotFound(err) {
		networkPolicy, err = clientset.NetworkingV1().NetworkPolicies(namespace).Create(context.TODO(), &v1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{Name: "netdns-policy-generated"}}, metav1.CreateOptions{})

		if err != nil {
			log.Println("Error creating policy: " + err.Error())
			return
		}

	} else if err != nil {
		log.Println("Error getting policy: " + err.Error())
		return
	}

	networkPolicy.Spec = v1.NetworkPolicySpec{
		Egress: []v1.NetworkPolicyEgressRule{
			{To: to}},
		PodSelector: settings.PodSelector}

	update := false

	if !reflect.DeepEqual(networkPolicy.Spec.Egress, networkPolicy2.Spec.Egress) {
		log.Println("LOG: DNS change, updating NetworkPolicy")
		update = true
	}

	if !reflect.DeepEqual(networkPolicy.Spec.PodSelector, networkPolicy2.Spec.PodSelector) {
		log.Println("LOG: PodSelector change, updating NetworkPolicy")
		update = true
	}

	if update {
		_, err = clientset.NetworkingV1().NetworkPolicies(namespace).Update(context.TODO(), networkPolicy, metav1.UpdateOptions{})

		if err != nil {
			log.Println("Error updating policy:" + err.Error())
		}
	}
}
