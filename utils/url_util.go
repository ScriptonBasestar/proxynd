package utils

import (
	"fmt"
	"path"
	"strings"
)

//func JoinURL1(host string, paths ... string) string {
//    u, err := url.Parse(host)
//    if err != nil {
//        panic(err)
//    }
//    u.Path = path.Join(u.Path, path.Join(paths...))
//    s := u.String()
//    return s
//}

func JoinURL(base string, paths ...string) string {
	p := path.Join(paths...)
	return fmt.Sprintf("%s/%s", strings.TrimRight(base, "/"), strings.TrimLeft(p, "/"))
}
