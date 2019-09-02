package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	FS_PREFIX     = "/fs"
	STABLE_PREFIX = "/us" // uniform speed
	MB            = 1 << 20
)

/** Descrption: 测试源端口与目的端口是否一致
 *  CreateTime: 2018/11/05 20:53:03
 *      Author: zhoutong@genomics.cn
 */
func main() {
	port := flag.String("p", "12345", "port")
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
		fmt.Printf("request time: %v\n\n", time.Since(start))
	}(time.Now())
	fmt.Printf("remote address: %v, url: %v\n", request.RemoteAddr, request.URL)
	if len(request.Cookies()) > 0 {
		fmt.Printf("cookies: %v\n", request.Cookies())
	}

	buf, _ := ioutil.ReadAll(request.Body)
	if len(buf) > 0 {
		fmt.Printf("body(length: %v):\n%v\n", len(buf), string(buf))
	}

	if strings.HasPrefix(request.URL.Path, FS_PREFIX) {
		filesystem(response, request)
		return
	} else if strings.HasPrefix(request.URL.Path, STABLE_PREFIX) {
		StableSpeedWrite(response, request)
		return
	}
	response.Write([]byte("ok\n"))
}

func filesystem(response http.ResponseWriter, request *http.Request) {
	path := request.URL.Path[len(FS_PREFIX):]
	// fmt.Printf("filepath: %v\n", path)
	f, err := os.Open(path)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer f.Close()
	http.ServeContent(response, request, path, time.Now(), f)
	return
}

func StableSpeedWrite(response http.ResponseWriter, request *http.Request) {
	var path = request.URL.Path[len(STABLE_PREFIX):]
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Println(err)
		return
	}

	querys := request.URL.Query()
	interval, _ := time.ParseDuration(querys.Get("interval"))
	if interval == 0 {
		interval = time.Second
	}
	report_interval, _ := time.ParseDuration(querys.Get("report"))
	if report_interval == 0 {
		report_interval = 5 * time.Second
	}

	speed, _ := strconv.ParseFloat(querys.Get("speed"), 0)
	if speed == 0 {
		speed = 10
	}
	block_size := int(speed * MB)
	slient := querys.Get("silent") == "true"
	nocycle := querys.Get("cycle") != "true"

	ticker := time.NewTicker(interval)
	rtikcer := time.NewTicker(report_interval)
	total, sent := len(buf), 0
	// var start time.Time
	start := time.Now()
	for {
		var block []byte
		end := sent + block_size
		if end > total {
			block = buf[sent:]
			if !nocycle {
				end -= total
				for end > total {
					block = append(block, buf...)
					end -= total
				}
				block = append(block, buf[:end]...)
			}
		} else {
			block = buf[sent:end]
		}

		select {
		case <-ticker.C:

			var remain = len(block)
			for remain > 0 {
				n, err := response.Write(block)
				if err != nil {
					fmt.Println(err)
					return
				}
				remain -= n
			}

			sent += (block_size - remain)
			if sent >= total {
				dur := time.Since(start)
				fmt.Printf("%.3fMBps, dur: %v\n", float64(sent)/(float64(dur.Nanoseconds())/1000), dur)
				if nocycle {
					return
				}
				sent = 0
				start = time.Now()
			}

		case <-rtikcer.C:
			if sent == 0 || slient {
				continue
			}

			dur := time.Since(start)
			fmt.Printf("%.3fMBps, dur: %v\n", float64(sent)/(float64(dur.Nanoseconds())/1000), dur)
		}
	}
}
