package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"math"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/ZoneTong/toolkits/common"
)

var (
	host       = flag.String("h", "127.0.0.1", "host ip")
	iface      = flag.String("I", "", "src net interface to dial")
	port       = flag.Int("p", 9999, "port")
	udp        = flag.Bool("u", false, "udp or tcp")
	print      = flag.Bool("v", false, "print")
	timeout    = flag.Duration("t", 5*time.Second, "dial time out")
	concurrent = flag.Int("c", 8, "cocurrent")
	interval   = flag.Duration("i", time.Second, "interval")

	done        = make(chan int)
	group       sync.WaitGroup
	summaryDone = make(chan int)
	midprint    = make(chan bool)
	sysch       = make(chan os.Signal, 1)

	ttlch           = make(chan WroteInfo)
	speed           = flag.Float64("speed", 10, "uniform speed")
	report_interval = flag.Duration("report", time.Second*5, "report interval")
	file            = flag.String("f", "", "file path")
)

const (
	MB = 1 << 20
)

type WroteInfo struct {
	size int
	dur  time.Duration
}

func main() {
	rand.Seed(time.Now().Unix())
	// fmt.SetFlags(fmt.LstdFlags | fmt.Lshortfile)
	flag.Parse()
	fmt.Println("pid:", os.Getpid())

	ticker := time.NewTicker(*interval / time.Duration(*concurrent))
	for i := 0; i < *concurrent; i++ {
		select {
		case <-ticker.C:
			conn, err := common.Dial(*iface, *host, *port, *timeout, *udp)
			if err != nil {
				return
			}

			go writeSpeed(conn)
		}
	}

	ticker.Stop()
	go stat()

	signal.Notify(sysch, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

LISTEN:
	select {
	case sig := <-sysch:
		switch sig {
		case syscall.SIGHUP:
			midprint <- true
			goto LISTEN

		default:
			close(done)
			for range sysch {
			}
		}

	}
	<-summaryDone
}

func stat() {

	var min, max = 10000.0, 0.0
	var total WroteInfo

	process := func(info WroteInfo) {
		writeSpeed := float64(info.size) / (float64(info.dur.Nanoseconds()) / 1000)
		if *print {
			fmt.Printf("writeSpeed: %.3fMBps\n", writeSpeed)
		}

		if math.Max(min, writeSpeed) == min {
			min = writeSpeed
		} else if math.Max(writeSpeed, max) == writeSpeed {
			max = writeSpeed
		}
		total.size += info.size
		total.dur += info.dur
	}
	for {
		select {
		case <-done:
			go func() {
				group.Wait()
				close(ttlch)
				close(sysch)
			}()

			for writeSpeed := range ttlch {
				process(writeSpeed)
			}
			fmt.Printf("writeSpeed min/avg/max(MBps): %.3f/%.3f/%.3f\n", min, float64(total.size)/(float64(total.dur.Nanoseconds())/1000), max)
			summaryDone <- 1
			return

		case writeSpeed := <-ttlch:
			process(writeSpeed)
		case <-midprint:
			fmt.Printf("writeSpeed min/avg/max(MBps): %.3f/%.3f/%.3f\n", min, float64(total.size)/(float64(total.dur.Nanoseconds())/1000), max)
		}
	}
}

func writeSpeed(conn net.Conn) {
	group.Add(1)
	defer group.Done()
	defer conn.Close()

	buf, err := ioutil.ReadFile(*file)
	if err != nil {
		fmt.Println(err)
		return
	}

	block_size := int(*speed * MB)
	nocycle := false
	ticker := time.NewTicker(*interval)
	total, sent := len(buf), 0
	var start time.Time
	for {
		start = time.Now()
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
		case <-done:
			return
		case <-ticker.C:
			var remain = len(block)
			for remain > 0 {
				n, err := conn.Write(block)
				if err != nil {
					fmt.Println(err)
					return
				}
				remain -= n
			}

			dur := time.Since(start)

			ttlch <- WroteInfo{block_size - remain, dur}
			// ttlch <- (float64(sent) / (float64(dur.Nanoseconds()) / 1000))
			sent += (block_size - remain)
			if sent >= total {
				if nocycle {
					return
				}
				sent = 0
				start = time.Now()
			}
		}
	}

	// close(done)
	// sysch <- syscall.SIGUSR2
}
