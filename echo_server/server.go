package main

import (
	"flag"
	"fmt"
	"log"
	"net"
)

var (
	host    = flag.String("h", "0.0.0.0", "host ip")
	port    = flag.Int("p", 12345, "port")
	reverse = flag.Bool("r", true, "reverse")
	udp     = flag.Bool("u", false, "udp or tcp")
)

func main() {
	flag.Parse()
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

	data := make([]byte, 1024)
	for {
		n, remoteAddr, err := listener.ReadFromUDP(data)
		if err != nil {
			log.Println(err)
		}

		log.Printf("<%s> %s\n", remoteAddr, data[:n])
		var out_data = reverseData(data, n, *reverse)
		_, err = listener.WriteToUDP(out_data, remoteAddr)
		if err != nil {
			log.Println(err)
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

	data := make([]byte, 1024)
	for {
		// conn.SetDeadline(time.Now().Add(time.Second * 10))
		n, err := conn.Read(data)
		// if err == io.EOF || io.e {
		// 	break
		// } else
		if err != nil {
			log.Println(err)
			break
		}

		fmt.Printf("<%s> %s\n", conn.RemoteAddr(), data[:n])
		var out_data = reverseData(data, n, *reverse)
		_, err = conn.Write(out_data)
		if err != nil {
			log.Println(err)
		}
	}
}

func reverseData(data []byte, n int, reverse bool) (out []byte) {
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
