package helpers

import (
	"gopkg.in/yaml.v3"
	"log"
	"os"
)

func ReadYaml(path string, out interface{}) {
	// cache - no expire
	yamlFile, err := os.ReadFile(path)
	if err != nil {
		log.Printf("read configs err #%v ", err)
	}
	log.Printf("read yaml file path: %s", path)
	err = yaml.Unmarshal(yamlFile, out)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
		panic(err)
	}
	log.Printf("read config success %s \n", out)
}
