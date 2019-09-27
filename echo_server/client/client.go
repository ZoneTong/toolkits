package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
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
	iface      = flag.String("I", "", "src net interface to dial")
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
	sysch       = make(chan os.Signal, 1)
)

func main() {
	rand.Seed(time.Now().Unix())
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	flag.Parse()
	log.Println("pid:", os.Getpid())
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

	signal.Notify(sysch, syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR1, syscall.SIGUSR2)

LISTEN:
	select {
	case sig := <-sysch:
		switch sig {
		case syscall.SIGUSR1:
			midprint <- true
			goto LISTEN

		default:
			close(done)
			for range sysch {
			}
		}
		// case <-done:
		// case <-summaryDone:
	}
	<-summaryDone
}

func stat() {
	// group.Add(1)
	// defer group.Done()

	process := func(rtt time.Duration) {
		if *ping && *print {
			log.Printf("rtt: %.3fms\n", rtt.Seconds()*1000)
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
				close(sysch)
			}()

			for rtt := range ttlch {
				process(rtt)
			}
			log.Printf("rtt min/avg/max(Millisecond): %.3f/%.3f/%.3f, loss/total: %v/%v\n", min.Seconds()*1000, total_dur.Seconds()*1000/float64(cnt), max.Seconds()*1000, cnt-recvd, cnt)
			summaryDone <- 1
			return
		case rtt := <-ttlch:
			process(rtt)
		case <-midprint:
			log.Printf("rtt min/avg/max(Millisecond): %.3f/%.3f/%.3f, loss/total: %v/%v\n", min.Seconds()*1000, total_dur.Seconds()*1000/float64(cnt), max.Seconds()*1000, cnt-recvd-int32(len(ttlch)), cnt)
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
	var network = "tcp"
	if udp {
		network = "udp"
	}
	if *iface == "" {
		conn, err = net.DialTimeout(network, fmt.Sprintf("%v:%v", *host, *port), *timeout)
	} else {
		switch network {
		case "tcp":
			var laddr, raddr *net.TCPAddr
			laddr, err = net.ResolveTCPAddr(network, *iface+":0")
			if err != nil {
				goto END
			}
			raddr, err = net.ResolveTCPAddr(network, fmt.Sprintf("%v:%v", *host, *port))
			if err != nil {
				goto END
			}

			conn, err = net.DialTCP(network, laddr, raddr)

		case "udp":
			var laddr, raddr *net.UDPAddr
			laddr, err = net.ResolveUDPAddr(network, *iface+":0")
			if err != nil {
				goto END
			}
			raddr, err = net.ResolveUDPAddr(network, fmt.Sprintf("%v:%v", *host, *port))
			if err != nil {
				goto END
			}

			conn, err = net.DialUDP(network, laddr, raddr)

		default:
			err = errors.New(network + " is not supported")
		}
	}

END:
	if err != nil {
		log.Println("dial error:", err)
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
		atomic.AddInt32(&cnt, 1)
		if err != nil {
			log.Println(err)
			if *ping {
				continue
			}
			break
		}

		conn.SetReadDeadline(time.Now().Add(*timeout))
		var n2, n int
		var rtt time.Duration

		// LOOP:
		for n1 > n2 {
			n, err = conn.Read(out[n2:])
			rtt = time.Since(since)
			n2 += n
			if err != nil {
				log.Println(n, err)
				break
			}
		}

		// conn.Close()
		if !reflect.DeepEqual(in[:n1], out[:n2]) {
			log.Println(n1, n2, string(in[:n1]), ", ", string(out[:n2]))
			if *ping {
				continue
			}
			break
		}

		if *print {
			log.Printf("%s(in len: %v), %s(out len: %v)\n", string(in[:n1]), n1, string(out[:n2]), n2)

			log.Printf("%s<-%s\n", conn.LocalAddr(), conn.RemoteAddr())
		}

		ttlch <- rtt

	}
	// close(done)
	sysch <- syscall.SIGUSR2
}
