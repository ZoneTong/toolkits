package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/ZoneTong/toolkits/common"
)

var (
	iface      = flag.String("I", "", "src net interface to dial")
	host       = flag.String("h", "0.0.0.0", "host ip")
	port       = flag.Int("p", 9999, "port")
	udp        = flag.Bool("u", false, "udp or tcp")
	word       = flag.String("w", "", "word")
	print      = flag.Bool("v", false, "print")
	detail     = flag.Bool("detail", false, "print detail")
	timeout    = flag.Duration("t", 2*time.Second, "dial time out")
	concurrent = flag.Int("c", 1, "cocurrent")
	cycle      = flag.Bool("cycle", false, "Cyclic write and read")
	interval   = flag.Duration("i", time.Second, "interval")
	sockbuf    = flag.Int("sockbuf", 1<<20, "sockbuf size")
	queuebuf   = flag.Int("buf", 1<<10, "socket queue size")

	pool = &sync.Pool{
		New: func() interface{} {
			return make([]byte, *sockbuf)
		}}

	wg sync.WaitGroup
)

func main() {
	// log.SetFlags(log.LstdFlags | log.Lshortfile)
	flag.Parse()

	if *word == "" {
		fmt.Printf("Input a word: ")
		fmt.Scanln(word)
	}

	initdata := []byte(*word)
	ticker := time.NewTicker(*interval / time.Duration(*concurrent))
	for i := 0; i < *concurrent; i++ {
		queue := make(chan []byte, *queuebuf)
		queue <- initdata

		select {
		case <-ticker.C:
			conn, err := common.Dial(*iface, *host, *port, *timeout, *udp)
			defer conn.Close()
			if err != nil {
				fmt.Println(err)
				return
			}

			wg.Add(2)
			go readLoop(conn, queue)
			go writeLoop(conn, queue)
		}
	}

	wg.Wait()
}

func writeLoop(conn net.Conn, dataque <-chan []byte) {
	defer wg.Done()
	for data := range dataque {
		var head int
		for head < len(data) {
			conn.SetWriteDeadline(time.Now().Add(*timeout))
			n, err := conn.Write(data[head:])
			head += n
			if err != nil {
				log.Println("write", head, len(data)-head, len(dataque), err)
				break
			}
		}
	}
}

func readLoop(conn net.Conn, dataque chan<- []byte) {
	defer wg.Done()
	defer close(dataque)
	for {
		var buf = pool.Get().([]byte)
		// conn.SetReadDeadline(time.Now().Add(*timeout))
		n, err := conn.Read(buf)
		if n > 0 {
			if *print {
				fmt.Printf("read data len: %v, queue len: %v\n", n, len(dataque))
				if *detail {
					fmt.Printf("%s\n", buf[:n])
				}
			}
			if *cycle {
				select {
				case dataque <- buf[:n]:
				default: // 这里不能阻塞, 因为要保证不断读,释放客户端滑动窗口
				}
			}
		} else {
			time.Sleep(*interval)
		}

		if err != nil {
			fmt.Println(n, err)
			return
		}

	}
}
