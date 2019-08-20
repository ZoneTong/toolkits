package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"reflect"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

var (
	host       = flag.String("h", "127.0.0.1", "host ip")
	port       = flag.Int("p", 9999, "port")
	udp        = flag.Bool("u", false, "udp or tcp")
	word       = flag.String("w", "", "word")
	wordlen    = flag.Int("l", 56, "word length")
	print      = flag.Bool("v", false, "print")
	timeout    = flag.Duration("t", 5*time.Second, "dial time out")
	concurrent = flag.Int("c", 8, "cocurrent")
	interval   = flag.Duration("i", time.Second, "ping interval")
	step       = flag.Int("step", 0, "word length grow by step")
	ping       = flag.Bool("ping", false, "ping to get rtt")
	maxPow     = flag.Uint("max", 0, "max_size = 2 ^ max")
	max_size   = 1 << 20
	// parall   chan bool

	done        = make(chan int)
	ttlch       = make(chan time.Duration)
	cnt, recvd  int32
	total_dur   time.Duration
	min, max    = time.Minute, time.Duration(0)
	group       sync.WaitGroup
	summaryDone = make(chan int)
	midprint    = make(chan bool)
)

func main() {
	rand.Seed(time.Now().Unix())
	// fmt.SetFlags(fmt.LstdFlags | fmt.Lshortfile)
	flag.Parse()
	fmt.Println("pid:", os.Getpid())
	if *maxPow != 0 {
		max_size = 1 << *maxPow
	}

	// if *word == "" {
	// 	*word = string(genstring(*wordlen))
	// }

	ticker := time.NewTicker(*interval / time.Duration(*concurrent))

	for i := 0; i < *concurrent; i++ {
		select {
		case <-ticker.C:
			conn, err := dial(*udp)
			if err != nil {
				return
			}

			go rtt(conn)
		}
	}

	ticker.Stop()
	go stat()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR1)

LISTEN:
	select {
	case sig := <-ch:
		switch sig {
		case syscall.SIGUSR1:
			midprint <- true
			goto LISTEN
		default:
			close(done)
		}
	case <-done:
	case <-summaryDone:
	}
	<-summaryDone
}

func stat() {
	// group.Add(1)
	// defer group.Done()

	process := func(rtt time.Duration) {
		if *ping {
			fmt.Printf("rtt: %.3fms\n", rtt.Seconds()*1000)
		}

		recvd++
		if rtt < min {
			min = rtt
		} else if rtt > max {
			max = rtt
		}
		total_dur += rtt
	}
	for {
		select {
		case <-done:
			go func() {
				group.Wait()
				close(ttlch)
			}()

			for rtt := range ttlch {
				process(rtt)
			}
			fmt.Printf("rtt min/avg/max(Millisecond): %.3f/%.3f/%.3f, loss/total: %v/%v\n", min.Seconds()*1000, total_dur.Seconds()*1000/float64(cnt), max.Seconds()*1000, cnt-recvd, cnt)
			summaryDone <- 1
			return
		case rtt := <-ttlch:
			process(rtt)
		case <-midprint:
			fmt.Printf("rtt min/avg/max(Millisecond): %.3f/%.3f/%.3f, loss/total: %v/%v\n", min.Seconds()*1000, total_dur.Seconds()*1000/float64(cnt), max.Seconds()*1000, cnt-recvd-int32(len(ttlch)), cnt)
		}
	}
}

func genstring(l int) (bs []byte) {
	// tstr := []byte(time.Now().Format("15:04:05.000000 2006/01/02 "))
	// if l <= len(tstr) {
	// 	return tstr[:l]
	// }

	// l -= len(tstr)
	chars := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	char_len := len(chars)
	for l > 0 {
		bs = append(bs, chars[rand.Intn(char_len)])
		l--
	}
	return bs
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

func rtt(conn net.Conn) {
	group.Add(1)
	defer group.Done()
	defer conn.Close()

	tick := time.NewTicker(*interval)

	length := *wordlen - *step
	var in = genstring(length)
	var out = make([]byte, max_size)
	for length < max_size {
		select {
		case <-tick.C:
		case <-done:
			return
		}

		in = append(in, genstring(*step)...)
		length = len(in)

		conn.SetDeadline(time.Now().Add(*timeout))
		since := time.Now()
		n1, err := conn.Write(in)
		if err != nil {
			fmt.Println(err)
			break
		}

		// conn.SetReadDeadline(time.Now().Add(*timeout))
		var n2, n int
		var rtt time.Duration
		atomic.AddInt32(&cnt, 1)

		// LOOP:
		for n1 > n2 {
			n, err = conn.Read(out[n2:])
			rtt = time.Since(since)
			n2 += n
			if err != nil {
				fmt.Println(n, err)
				break
			}
		}

		// conn.Close()
		if !reflect.DeepEqual(in[:n1], out[:n2]) {
			fmt.Println(n1, n2, string(in[:n1]), ", ", string(out[:n2]))
			break
		}

		if *print {
			fmt.Printf("%s(in len: %v), %s(out len: %v)\n", string(in[:n1]), n1, string(out[:n2]), n2)
		}

		ttlch <- rtt

	}
	close(done)
}
