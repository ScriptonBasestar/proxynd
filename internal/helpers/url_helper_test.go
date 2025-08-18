package helpers

import (
	"fmt"
	"testing"
)

//    result1 := JoinURL("http://naver.com/", "/download")
//    fmt.Println(result1)
//}

func TestJoinURL(_ *testing.T) {
	result1 := JoinURL("http://naver.com/", "/download")
	fmt.Println(result1)
}
