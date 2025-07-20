package helpers

import (
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

// ReadYaml YAML 파일을 읽습니다 (하위 호환성을 위해 유지, 에러 시 log.Fatal)
// 새로운 코드에서는 ReadYamlSafe 사용을 권장합니다
func ReadYaml(path string, out interface{}) {
	yamlFile, err := os.ReadFile(path)
	if err != nil {
		log.Printf("read configs err #%v ", err)
		log.Fatalf("Failed to read YAML file %s: %v", path, err)
		return
	}
	log.Printf("read yaml file path: %s", path)
	err = yaml.Unmarshal(yamlFile, out)
	if err != nil {
		log.Fatalf("Failed to unmarshal YAML file %s: %v", path, err)
		return
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

// ToStringYaml is exported
// ToStringYaml converts data between formats
func ToStringYaml(out interface{}) string {
	d, err := yaml.Marshal(out)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	log.Printf("---\n%s\n", string(d))
	return string(d)
}
