package main

import (
	"flag"
	"fmt"
	"log"
	"net"
)

var (
	host     = flag.String("h", "0.0.0.0", "host ip")
	port     = flag.Int("p", 12345, "port")
	reverse  = flag.Bool("r", true, "reverse")
	udp      = flag.Bool("u", false, "udp or tcp")
	noprint  = flag.Bool("z", false, "no print")
	multiple = flag.Int("m", 1, "multiple times echo back")
	max_size = 1 << 20

	limit  = flag.Int("l", 1024, "limit")
	parall chan bool
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

	data := make([]byte, 1024)
	for {
		n, remoteAddr, err := listener.ReadFromUDP(data)
		if err != nil {
			log.Println(err)
		}

		if !*noprint {
			log.Printf("<%s> %s\n", remoteAddr, data[:n])
		}

		<-parall
		go echoUDP(listener, remoteAddr, data, n)
	}
}

func echoUDP(listener *net.UDPConn, remoteAddr *net.UDPAddr, data []byte, n int) {
	var out_data = multipleData(reverseData(data[:n], *reverse))
	_, err := listener.WriteToUDP(out_data, remoteAddr)
	if err != nil {
		log.Println(err)
	}

	parall <- true
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
		<-parall
		n, err := conn.Read(data)
		// if err == io.EOF || io.e {
		// 	break
		// } else
		if err != nil {
			log.Println(err)
			break
		}

		if !*noprint {
			fmt.Printf("<%s>%s(parall: %v)\n", conn.RemoteAddr(), data[:n], *limit-len(parall))
		}

		go echoTCP(conn, data[:n])
	}
}

func echoTCP(conn net.Conn, data []byte) {
	var out_data = multipleData(reverseData(data, *reverse))
	_, err := conn.Write(out_data)
	if err != nil {
		log.Println(err)
	}

	parall <- true
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

func multipleData(data []byte) (out []byte) {
	if len(data) >= max_size {
		return data[:max_size]
	}

	for i := 0; i < *multiple; i++ {
		out = append(out, data...)
	}
	return
}
