package main

import (
	"flag"
	"fmt"
	"net"
	"sync/atomic"
)

var (
	host = flag.String("h", "0.0.0.0", "host ip")
	port = flag.Int("p", 9999, "port")

	udp      = flag.Bool("u", false, "udp or tcp")
	print    = flag.Bool("v", false, "print")
	multiple = flag.Int("m", 1, "multiple times echo back, negtive means to reverse")
	step     = flag.Int("step", 0, "positive means shift left, negtive means shift right, when multiple is not -1/0/1")
	// limit    = flag.Int("l", 1024, "limit")

	max_size = 1024
	reverse  bool
	// parall   chan bool
	parallCount int32
)

func main() {
	flag.Parse()
	if *multiple < 0 {
		*multiple = -*multiple
		reverse = true
	}

	// parall = make(chan bool, *limit)
	// for i := 0; i < *limit; i++ {
	// 	parall <- true
	// }

	if *udp {
		listenUDP()
	} else {
		listenTCP()
	}
}

func listenUDP() {
	listener, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP(*host), Port: *port})
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("listen at udp:%v:%v\n", *host, *port)

	data := make([]byte, max_size)
	for {
		n, remoteAddr, err := listener.ReadFromUDP(data)
		if err != nil {
			fmt.Println(err)
		}

		// <-parall
		go echoBack(data[:n], func(out []byte) (int, error) {
			return listener.WriteToUDP(out, remoteAddr)
		})

		if *print {
			fmt.Printf("<%s>%s(para: %v)\n", remoteAddr, data[:n], parallCount)
		}

	}
}

func echoBack(data []byte, f func([]byte) (int, error)) {
	atomic.AddInt32(&parallCount, 1)
	defer func() { atomic.AddInt32(&parallCount, -1) }()
	var out_data = reverseData(data)
	var distance = *step
	if len(out_data) < 2 {
		distance = 0
	}
	for distance < 0 {
		distance += len(out_data)
	}
	for i := 0; i < *multiple; i++ {
		out_data = append(out_data[distance:], out_data[:distance]...)
		// fmt.Println(string(out_data))
		_, err := f(out_data)
		if err != nil {
			fmt.Println(err)
			return
		}
	}
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

	data := make([]byte, max_size)
	for {
		// <-parall
		n, err := conn.Read(data)
		if err != nil {
			fmt.Println(err)
			break
		}

		if *print {
			fmt.Printf("<%s>%s(parall: %v)\n", conn.RemoteAddr(), data[:n], parallCount)
		}

		go echoBack(data[:n], func(out []byte) (int, error) {
			return conn.Write(out)
		})
	}
}

func reverseData(data []byte) (out []byte) {
	if !reverse {
		return data
	}

	n := len(data)
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

func multipleData(data []byte) (out []byte) {
	if len(data) >= max_size {
		return data[:max_size]
	}

	for i := 0; i < *multiple; i++ {
		out = append(out, data...)
	}
	return
}
