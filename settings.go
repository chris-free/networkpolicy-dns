package main

import (
	"errors"
	"io/ioutil"
	"log"

	"github.com/ghodss/yaml"
	"gopkg.in/fsnotify.v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func watchSettings(reset chan<- bool, settingsPath string) {
	go func() {
		watcher, err := fsnotify.NewWatcher()
		defer watcher.Close()

		if err != nil {
			log.Fatal(err)
		}

		err = watcher.Add(settingsPath)
		if err != nil {
			log.Fatal(err)
		}

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					log.Fatal("Settings watch channel closed")
					return
				}
				log.Println("event:", event)
				if event.Op&fsnotify.Write == fsnotify.Write {
					log.Println("Settings modified file:", event.Name)
					reset <- true
				}
			case err, _ := <-watcher.Errors:
				log.Fatal("Settings watch error:", err)
				return
			}
		}

	}()
}

type Settings struct {
	Domain      []string             `json:"domain"`
	PodSelector metav1.LabelSelector `json:"podSelector"`
	Interval    int                  `json:"interval"`
}

func readSettings(settingsPath string) (Settings, error) {
	settings := Settings{
		Interval: 300,
	}

	settingsBytes, err := ioutil.ReadFile(settingsPath)

	if err != nil {
		return Settings{}, errors.New("Error opening settings: " + err.Error())
	}

	err = yaml.Unmarshal(settingsBytes, &settings)

	if err != nil {
		return Settings{}, errors.New("Error unmarshalling settings: " + err.Error())
	}

	return settings, nil
}

func getInterval(settingsPath string) int {
	settings, err := readSettings(settingsPath)

	if err != nil {
		log.Fatal(err)
	}

	return settings.Interval
}
