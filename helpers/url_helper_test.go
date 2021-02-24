package helpers

import (
	"fmt"
	"testing"
)

//func TestJoinURL1(t *testing.T) {
//    result1 := JoinURL("http://naver.com/", "/download")
//    fmt.Println(result1)
//}

func TestJoinURL(t *testing.T) {
	result1 := JoinURL("http://naver.com/", "/download")
	fmt.Println(result1)
}
