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
		panic(err)
	}
	log.Printf("read yaml file path: %s", path)
	err = yaml.Unmarshal(yamlFile, out)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
		panic(err)
	}
	log.Printf("read config success %s \n", out)
}

// ReadYamlSafe YAML 파일을 안전하게 읽습니다 (에러 반환)
func ReadYamlSafe(path string, out interface{}) error {
	yamlFile, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(yamlFile, out)
	if err != nil {
		return err
	}

	return nil
}

// WriteYaml YAML 파일을 작성합니다
func WriteYaml(path string, data interface{}) error {
	yamlData, err := yaml.Marshal(data)
	if err != nil {
		return err
	}

	return os.WriteFile(path, yamlData, 0644)
}

func ToStringYaml(out interface{}) string {
	d, err := yaml.Marshal(out)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	log.Printf("---\n%s\n", string(d))
	return string(d)
}
