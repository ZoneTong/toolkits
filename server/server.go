package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

/** Descrption: 测试源端口与目的端口是否一致
 *  CreateTime: 2018/11/05 20:53:03
 *      Author: zhoutong@genomics.cn
 */
func main() {
	port := flag.String("p", "9999", "port")
	flag.Parse()
	fmt.Printf("listen at :%v\n", *port)

	log.Println(http.ListenAndServe(":"+*port, &myHandler{}))
}

type myHandler struct {
}

/** Descrption:
 *  CreateTime: 2018/11/05 20:56:55
 *      Author: zhoutong@genomics.cn
 */
func (h myHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	defer func(start time.Time) {
		fmt.Println(time.Since(start))
	}(time.Now())
	fmt.Printf("remote addr: %v\n", request.RemoteAddr)
	fmt.Printf("url: %v\n", request.URL)
	buf, _ := ioutil.ReadAll(request.Body)
	fmt.Printf("body(len: %v): %v\n\n", len(buf), string(buf))

	if strings.HasPrefix(request.URL.Path, "/fs") {
		path := request.URL.Path[4:]
		fmt.Printf("filepath: %v\n", path)
		f, err := os.Open(path)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer f.Close()
		http.ServeContent(response, request, path, time.Now(), f)
		return
	}
	response.Write([]byte("ok\n"))
}
