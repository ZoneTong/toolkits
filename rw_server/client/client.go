package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"time"
)

var (
	host       = flag.String("h", "0.0.0.0", "host ip")
	port       = flag.Int("p", 12345, "port")
	udp        = flag.Bool("u", false, "udp or tcp")
	word       = flag.String("w", "", "word")
	noprint    = flag.Bool("z", false, "no print")
	limit      = flag.Int("l", 1024, "limit")
	timeout    = flag.Duration("t", 2*time.Second, "dial time out")
	concurrent = flag.Int("c", 10, "cocurrent")
	max_size   = 1 << 20
	parall     chan bool
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	flag.Parse()
	parall = make(chan bool, *limit)
	for i := 0; i < *limit; i++ {
		parall <- true
	}

	if *word == "" {
		fmt.Printf("Input a word: ")
		fmt.Scanln(word)
	}
	for i := 0; i < *concurrent; i++ {
		if *udp {
			go udpDial()
		} else {
			go tcpDial()
		}
	}

	select {}
}

func tcpDial() {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%v:%v", *host, *port), *timeout)
	if err != nil {
		log.Println("dial error:", err)
		return
	}
	defer conn.Close()
	// conn.Write([]byte(*word))
	// go func() {
	// for {
	_, err = conn.Write([]byte(*word))
	if err != nil {
		log.Println(err)
		return
	}
	// }
	// }()

	var buf = make([]byte, max_size)
	for {
		// <-parall
		// conn.SetReadDeadline(time.Now().Add(*timeout))
		n, err := conn.Read(buf)
		if err != nil && err != io.EOF {
			// if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
			// continue
			// }
			return
		}

		if !*noprint {
			log.Printf("conn read %d bytes,  error: %s", n, err)
		}

		// conn.SetWriteDeadline(time.Now().Add(*timeout))
		// go echoTcp(conn, buf[:n])
	}
}

func udpDial() {
	conn, err := net.DialUDP("udp", nil, &net.UDPAddr{IP: net.ParseIP(*host), Port: *port})
	if err != nil {
		log.Fatalln("Can't dial: ", err)
		return
	}
	// log.Printf("%v dial succeeded!\n", conn.LocalAddr())
	defer conn.Close()
	var cnt int
	// go func() {
	// 	for {
	// 		cnt++
	_, err = conn.Write([]byte(*word))
	if err != nil {
		log.Printf("%v failed %v: %v\n", conn.LocalAddr(), cnt, err)
		// time.Sleep(time.Second)
		// return
	}
	// 	}
	// }()

	data := make([]byte, max_size)
	for {
		n, err := conn.Read(data)
		if err != nil && err != io.EOF {
			log.Printf("%v failed to read UDP msg because of %v\n", conn.LocalAddr(), err)
			return
		}
		if !*noprint {
			log.Println(string(data[:n]), err)
		}
	}
}
