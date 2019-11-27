package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	FS_PREFIX     = "/fs"
	STABLE_PREFIX = "/us" // uniform speed
	MB            = 1 << 20
)

var (
	server   *http.Server
	listener net.Listener
	rootpath = flag.String("root", "/", "filesystem root path")
)

/** Descrption: 测试源端口与目的端口是否一致
 *  CreateTime: 2018/11/05 20:53:03
 *      Author: zhoutong@genomics.cn
 */
func main() {
	port := flag.Int("p", 9999, "port")
	graceful := flag.Bool("update", false, "graceful update")
	flag.Parse()

	var err error
	if *graceful {
		f := os.NewFile(3, "")
		listener, err = net.FileListener(f)
		if err != nil {
			log.Fatal(err)
		}
		f.Close()
	} else {
		addr := ":" + fmt.Sprint(*port)
		listener, err = net.Listen("tcp", addr)
		if err != nil {
			log.Fatal(err)
		}
	}
	log.Printf("listen at %v\n", listener.Addr())

	server = &http.Server{Handler: http.HandlerFunc(myHandle)}
	go func() {
		if err := server.Serve(listener); err != nil {
			log.Println(err)
		}
	}()

	signalHandler()
}

/** Descrption:
 *  CreateTime: 2018/11/05 20:56:55
 *      Author: zhoutong@genomics.cn
 */
func myHandle(response http.ResponseWriter, request *http.Request) {
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
	path = filepath.Join(*rootpath, path)
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

func signalHandler() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR2)
	for {
		sig := <-ch
		log.Printf("signal: %v", sig)

		// timeout context for shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()
		switch sig {
		case syscall.SIGINT, syscall.SIGTERM:
			// stop
			log.Printf("stop")
			signal.Stop(ch)
			server.Shutdown(ctx)
			log.Printf("graceful shutdown")
			return

		case syscall.SIGUSR2:
			// reload
			log.Printf("reload")
			err := reload()
			if err != nil {
				log.Fatalf("graceful restart error: %v", err)
				continue
			}
			server.Shutdown(ctx)
			log.Printf("graceful reload")
			return
		}
	}
}

func reload() error {
	tl, ok := listener.(*net.TCPListener)
	if !ok {
		return errors.New("listener is not tcp listener")
	}

	f, err := tl.File()
	if err != nil {
		return err
	}

	args := []string{"-update"}
	cmd := exec.Command(os.Args[0], args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	// put socket FD at the first entry
	cmd.ExtraFiles = []*os.File{f}
	return cmd.Start()
}
