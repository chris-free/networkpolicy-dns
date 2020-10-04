package main

import (
	"context"
	"flag"
	"fmt"
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

var settingsPath *string

func main() {

	settingsPath = flag.String("settings", "/configmap/settings.yml", "Path to settings.yml")

	flag.Parse()

	var err error

	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatal(err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}

	reset := make(chan bool)

	watchSettings(reset, *settingsPath)

	ticker := newTicker()

	run(clientset)

	for {
		select {
		case <-reset:
			fmt.Println("Reset")
			ticker.Stop()
			ticker = newTicker()
		case <-ticker.C:
			log.Println("tick")
			run(clientset)
		}
	}
}

func newTicker() (newTicker *time.Ticker) {
	settings, err := readSettings(*settingsPath)

	if err != nil {
		log.Fatal(err)
	}

	return time.NewTicker(time.Duration(settings.Interval) * time.Second)
}

func run(clientset *kubernetes.Clientset) {

	settings, err := readSettings(*settingsPath)

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

	sort.Strings(addrs) // Required for Deep equal check

	var to []v1.NetworkPolicyPeer
	for _, addr := range addrs {
		to = append(to, v1.NetworkPolicyPeer{IPBlock: &v1.IPBlock{CIDR: addr + "/32"}})
	}

	var newPolicy *v1.NetworkPolicy

	var existingPolicy *v1.NetworkPolicy

	namespaceBytes, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")

	if err != nil {
		log.Println("Error reading namespace service account file")
		return
	}

	namespace := string(namespaceBytes)

	existingPolicy, err = clientset.NetworkingV1().NetworkPolicies(namespace).Get(context.TODO(), "netdns-policy-generated", metav1.GetOptions{})

	newPolicy := existingPolicy

	if apiErrors.IsNotFound(err) {
		newPolicy, err = clientset.NetworkingV1().NetworkPolicies(namespace).Create(context.TODO(), &v1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{Name: "netdns-policy-generated"}}, metav1.CreateOptions{})

		if err != nil {
			log.Println("Error creating policy: " + err.Error())
			return
		}

	} else if err != nil {
		log.Println("Error getting policy: " + err.Error())
		return
	}

	newPolicy.Spec = v1.NetworkPolicySpec{
		Egress: []v1.NetworkPolicyEgressRule{
			{To: to}},
		PodSelector: settings.PodSelector}

	update := false

	if !reflect.DeepEqual(newPolicy.Spec.Egress, existingPolicy.Spec.Egress) {
		log.Println("LOG: DNS change, updating NetworkPolicy")
		update = true
	}

	if !reflect.DeepEqual(newPolicy.Spec.PodSelector, existingPolicy.Spec.PodSelector) {
		log.Println("LOG: PodSelector change, updating NetworkPolicy")
		update = true
	}

	if update {
		_, err = clientset.NetworkingV1().NetworkPolicies(namespace).Update(context.TODO(), newPolicy, metav1.UpdateOptions{})

		if err != nil {
			log.Println("Error updating policy:" + err.Error())
		}
	}
}
