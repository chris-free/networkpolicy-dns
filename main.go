package main

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"reflect"
	"sort"
	"time"

	"github.com/ghodss/yaml"
	"gopkg.in/fsnotify.v1"
	v1 "k8s.io/api/networking/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type Settings struct {
	Domain      []string             `json:"domain"`
	PodSelector metav1.LabelSelector `json:"podSelector"`
}

// todo
// use log package
// 4. name
// get settings to share ticker/run
// 5. use: https://github.com/kubernetes/apimachinery/blob/master/pkg/util/yaml/decoder.go for unmarshalling settings

func watch(reset chan<- bool) {
	go func() {
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			log.Fatal(err)
		}
		defer watcher.Close()

		err = watcher.Add("/app/settings.yml")
		if err != nil {
			log.Fatal(err)
		}

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				log.Println("event:", event)
				if event.Op&fsnotify.Write == fsnotify.Write {
					log.Println("modified file:", event.Name)
					reset <- true
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}

	}()
}
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

	watch(reset)

	var ticker *time.Ticker
	ticker = time.NewTicker(10 * time.Second)

	for {
		select {
		case <-reset:
			ticker.Stop()
			ticker = time.NewTicker(500 * time.Millisecond)
		case <-ticker.C:
			fmt.Println("tick")
			run(clientset)
		}
	}
}

func readSettings() (Settings, error) {
	var settings Settings
	//settingsBytes, err := ioutil.ReadFile("/configmap/settings.yml")

	settingsBytes, err := ioutil.ReadFile("/app/settings.yml")

	if err != nil {
		fmt.Println("Error opening settings: " + err.Error())
		return Settings{}, errors.New("asd")
	}

	err = yaml.Unmarshal(settingsBytes, &settings)

	if err != nil {
		fmt.Println("Error unmarshalling settings: " + err.Error())
		return Settings{}, errors.New("asd")
	}

	return settings, nil
}

func run(clientset *kubernetes.Clientset) {

	settings, err := readSettings()

	if err != nil {
		return
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
	if apiErrors.IsNotFound(err) {
		networkPolicy, err = clientset.NetworkingV1().NetworkPolicies("default").Create(context.TODO(), &v1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{Name: "netdns-policy-generated"}}, metav1.CreateOptions{})

		if err != nil {
			fmt.Println("Error creating policy: " + err.Error())
			return
		}

	} else if err != nil {
		fmt.Println("Error getting policy: " + err.Error())
		return
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
}
