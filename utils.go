package main

import (
	"io/ioutil"
	"math"
	"math/rand"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Strings struct {
	SubscribeResponse   string `yaml:"sub_response"`
	UnsubscribeResponse string `yaml:"unsub_response"`
	DefaultResponse     string `yaml:"default_response"`
	Notification        string `yaml:"notification"`
}

func randomInt32() int32 {
	rand.Seed(time.Now().UnixNano())
	return rand.Int31n(math.MaxInt32)
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getStrings(path string) (Strings, error) {
	file, err := os.Open(path)
	if err != nil {
		return Strings{}, err
	}
	defer file.Close()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		return Strings{}, err
	}

	res := Strings{}
	if err := yaml.Unmarshal(data, &res); err != nil {
		return Strings{}, err
	}

	return res, nil
}
