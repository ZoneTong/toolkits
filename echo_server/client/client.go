package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

var (
	host       = flag.String("h", "0.0.0.0", "host ip")
	port       = flag.Int("p", 9999, "port")
	udp        = flag.Bool("u", false, "udp or tcp")
	word       = flag.String("w", "", "word")
	wordlen    = flag.Int("l", 56, "word length")
	print      = flag.Bool("v", false, "print")
	timeout    = flag.Duration("t", 2*time.Second, "dial time out")
	concurrent = flag.Int("c", 8, "cocurrent")
	interval   = flag.Duration("i", time.Second, "ping interval")

	max_size = 1024
	// parall   chan bool

	done        = make(chan int)
	ttlch       = make(chan time.Duration)
	cnt, recvd  int32
	total_dur   time.Duration
	min, max    = time.Minute, time.Duration(0)
	group       sync.WaitGroup
	summaryDone = make(chan int)
)

func main() {
	rand.Seed(time.Now().Unix())
	// fmt.SetFlags(fmt.LstdFlags | fmt.Lshortfile)
	flag.Parse()

	// if *word == "" {
	// 	*word = string(genstring(*wordlen))
	// }

	ticker := time.NewTicker(*interval / time.Duration(*concurrent))

	for i := 0; i < *concurrent; i++ {
		conn, err := dial(*udp)
		if err != nil {
			return
		}
		select {
		case <-ticker.C:
			go ttl(conn)
		}
	}

	ticker.Stop()
	go stat()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-ch:
		close(done)
	}
	<-summaryDone
}

func stat() {
	// group.Add(1)
	// defer group.Done()

	process := func(ttl time.Duration) {
		if *print {
			fmt.Printf("ttl: %.3fms\n", ttl.Seconds()*1000)
		}

		recvd++
		if ttl < min {
			min = ttl
		} else if ttl > max {
			max = ttl
		}
		total_dur += ttl
	}
	for {
		select {
		case <-done:
			group.Wait()
			close(ttlch)
			for ttl := range ttlch {
				process(ttl)
			}
			fmt.Printf("ttl min/avg/max(Millisecond): %.3f/%.3f/%.3f, loss/total: %v/%v\n", min.Seconds()*1000, total_dur.Seconds()*1000/float64(cnt), max.Seconds()*1000, cnt-recvd, cnt)
			summaryDone <- 1
			return
		case ttl := <-ttlch:
			process(ttl)
		}
	}
}

func genstring(l int) []byte {
	tstr := []byte(time.Now().Format("15:04:05.000000 2006/01/02 "))
	if l <= len(tstr) {
		return tstr[:l]
	}

	l -= len(tstr)
	chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMZOPQRSTUVWXYZ"
	char_len := len(chars)
	for l > 0 {
		tstr = append(tstr, chars[rand.Intn(char_len)])
		l--
	}
	return tstr
}

func dial(udp bool) (conn net.Conn, err error) {
	if udp {
		conn, err = net.DialUDP("udp", nil, &net.UDPAddr{IP: net.ParseIP(*host), Port: *port})
	} else {
		conn, err = net.DialTimeout("tcp", fmt.Sprintf("%v:%v", *host, *port), *timeout)
	}

	if err != nil {
		fmt.Println("dial error:", err)
		return
	}
	return
}

func ttl(conn net.Conn) {
	group.Add(1)
	defer group.Done()
	defer conn.Close()

	tick := time.NewTicker(*interval)

	var buf = make([]byte, max_size)
	for {
		select {
		case <-tick.C:
		case <-done:
			return
		}

		in := genstring(*wordlen)

		since := time.Now()
		_, err := conn.Write(in)
		if err != nil {
			fmt.Println(err)
			continue
		}

		conn.SetReadDeadline(time.Now().Add(*timeout))
		_, err = conn.Read(buf)
		if err != nil {
			fmt.Println(err)
			continue
		}
		ttl := time.Since(since)
		// if *print {
		// fmt.Printf("%s(len: %v)\n", string(buf[:n]), n)
		// }

		atomic.AddInt32(&cnt, 1)
		ttlch <- ttl

	}
}
