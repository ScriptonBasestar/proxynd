package helpers

import (
	"fmt"
	"testing"
)

func TestReadYaml(t *testing.T) {
	yaml := ReadYaml("maven-proxy.yaml")
	fmt.Println(yaml)
}
