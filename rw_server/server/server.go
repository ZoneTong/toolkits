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
	host     = flag.String("h", "0.0.0.0", "host ip")
	port     = flag.Int("p", 12345, "port")
	reverse  = flag.Bool("r", true, "reverse")
	udp      = flag.Bool("u", false, "udp or tcp")
	noprint  = flag.Bool("z", false, "no print")
	multiple = flag.Int("m", 1, "multiple times echo back")
	limit    = flag.Int("l", 1024, "limit")
	interval = flag.Duration("i", time.Millisecond, "write interval")
	max_size = 1 << 20
	parall   chan bool
)

func main() {
	flag.Parse()
	parall = make(chan bool, *limit)
	for i := 0; i < *limit; i++ {
		parall <- true
	}

	if *udp {
		listenUDP()
	} else {
		listenTCP()
	}
}

func listenUDP() {
	listener, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP(*host), Port: *port})
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Printf("listen at udp:%v:%v\n", *host, *port)

	var closed = make(chan bool)
	data := make([]byte, 1024)
	var remotes = make(map[string]bool)

	for {
		n, remoteAddr, err := listener.ReadFromUDP(data)
		if err != nil {
			log.Println(err)
			close(closed)
			break
		}

		if !*noprint {
			log.Printf("<%s> %s\n", remoteAddr, data[:n])
		}

		// <-parall
		_, ok := remotes[remoteAddr.String()]
		if !ok {
			go func(addr *net.UDPAddr) {
				ticker := time.NewTicker(*interval)
				for {
					select {
					case <-ticker.C:
					case <-closed:
						return
					}

					echoUDP(listener, addr, data)
				}
			}(remoteAddr)
			remotes[remoteAddr.String()] = true
		}
	}
}

func echoUDP(listener *net.UDPConn, remoteAddr *net.UDPAddr, data []byte) {
	var out_data = reverseData(data, *reverse)
	for i := 0; i < *multiple; i++ {
		_, err := listener.WriteToUDP(out_data, remoteAddr)
		if err != nil {
			log.Println(err)
		}
	}
	// parall <- true
}

func listenTCP() {
	l, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP(*host), Port: *port})
	if err != nil {
		fmt.Println("Error listening:", err)
		return
	}
	defer l.Close()
	fmt.Printf("listen at tcp:%v:%v\n", *host, *port)

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting: ", err)
			continue
		}

		go handle(conn)
	}
}

func handle(conn net.Conn) {
	defer conn.Close()

	var closed = make(chan bool)
	data := make([]byte, max_size)
	go func() {
		ticker := time.NewTicker(*interval)
		for {
			select {
			case <-ticker.C:
			case <-closed:
				return
			}

			echoTCP(conn, data)
		}
	}()

	for {
		// <-parall

		n, err := conn.Read(data)
		if err != nil && err != io.EOF {
			log.Println(err)
			close(closed)
			return
		}

		if !*noprint {
			fmt.Printf("<%s>%s(parall: %v)\n", conn.RemoteAddr(), data[:n], *limit-len(parall))
		}

		// go echoTCP(conn, data, n)
	}
}

func echoTCP(conn net.Conn, data []byte) {
	var out_data = (reverseData(data, *reverse))
	for i := 0; i < *multiple; i++ {
		_, err := conn.Write(out_data)
		if err != nil {
			log.Println(err)
		}
	}
	// parall <- true
}

func reverseData(data []byte, reverse bool) (out []byte) {
	n := len(data)
	if !reverse {
		return data[:n]
	}

	out = make([]byte, n)
	if data[n-1] == '\n' {
		out[n-1] = '\n'
		n--
	}

	for i := 0; i < n; i++ {
		out[i] = data[n-1-i]
	}
	return
}

// func multipleData(data []byte) (out []byte) {
// 	if len(data) >= max_size {
// 		return data[:max_size]
// 	}

// 	for i := 0; i < *multiple; i++ {
// 		out = append(out, data...)
// 	}
// 	return
// }

func multipleEcho(f func()) {
	for i := 0; i < *multiple; i++ {
		f()
	}
}
