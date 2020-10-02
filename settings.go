package main

import (
	"errors"
	"io/ioutil"
	"log"

	"github.com/ghodss/yaml"
	"gopkg.in/fsnotify.v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Settings struct {
	Domain      []string             `json:"domain"`
	PodSelector metav1.LabelSelector `json:"podSelector"`
	interval    int
}

func watchSettings(reset chan<- bool) {
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

func readSettings() (Settings, error) {
	settings := Settings{
		interval: 60,
	}

	settingsBytes, err := ioutil.ReadFile("/configmap/settings.yml")

	if err != nil {
		return Settings{}, errors.New("Error opening settings: " + err.Error())
	}

	err = yaml.Unmarshal(settingsBytes, &settings)

	if err != nil {
		return Settings{}, errors.New("Error unmarshalling settings: " + err.Error())
	}

	return settings, nil
}
