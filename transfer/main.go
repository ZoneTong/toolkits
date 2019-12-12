package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"net"
	"sync/atomic"
	"time"
)

var (
	listen = flag.String("l", ":19999", "listen")
	dst    = flag.String("d", "127.0.0.1:9999", "destination address")
	// src        = flag.String("s", "192.168.0.109:10555", "source address")
	network    = flag.String("net", "udp", "network")
	packetsize = flag.Int("size", 10240, "packet size")
	print      = flag.Bool("v", false, "print")
	uint       = flag.String("uint", "k", "bits uint: m,k,b ")

	bytesFromClient uint64
	bytesFromServer uint64
)

const BITS = 8
const (
	B = 1 << (10 * iota)
	K
	M
	G
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	flag.Parse()
	listen_conn, err := net.ListenPacket(*network, *listen)
	if err != nil {
		log.Fatal(err)
		return
	}
	dst_conn, err := net.Dial(*network, *dst)
	if err != nil {
		log.Fatal(err)
		return
	}

	log.Printf("listen at %v, to %v\n", *listen, *dst)

	// src_addr, err := net.ResolveUDPAddr(*network, *src)
	// if err != nil {
	// 	log.Fatal(err)
	// 	return
	// }

	var src_addr net.Addr
	go func() {
		buf2 := make([]byte, *packetsize)
		var total int
		for {
			n, err := dst_conn.Read(buf2)
			if n != 0 {
				if *print {
					log.Println("recv from dest", *dst, n)
				}
				atomic.AddUint64(&bytesFromServer, uint64(n))

				m, werr := listen_conn.WriteTo(buf2[:n], src_addr)
				if *print {
					log.Println("send to src", src_addr, m)
				}
				total += m
				if m != n || werr != nil {
					log.Println("listen_conn.WriteTo", n, m, werr, total)
				}
			}

			if err != nil {
				log.Println(err)
				continue
			}
		}
	}()

	go printSpeed()
	buf := make([]byte, *packetsize)
	var ltotal int
	for {
		var n int
		var err error
		n, src_addr, err = listen_conn.ReadFrom(buf)
		if n != 0 {
			if *print {
				log.Println("recv from src", src_addr, n)
			}
			atomic.AddUint64(&bytesFromClient, uint64(n))
			m, werr := dst_conn.Write(buf[:n])
			if *print {
				log.Println("send to dst", *dst, m)
			}
			ltotal += m
			if m != n || werr != nil {
				log.Println("listen_conn.WriteTo", n, m, werr, ltotal)
			}
		}

		if err != nil {
			log.Println(err)
			continue
		}
	}
}

func printSpeed() {
	ticker := time.NewTicker(time.Second)
	var oldserver, oldclient uint64
	for {
		select {
		case <-ticker.C:
			serverCount, clientCount := atomic.LoadUint64(&bytesFromServer), atomic.LoadUint64(&bytesFromClient)
			if clientCount == oldclient && serverCount == oldserver {
				continue
			}

			tmp := clientCount
			if clientCount < oldclient {
				clientCount += (math.MaxUint64 - oldclient)
			} else {
				clientCount -= oldclient
			}
			oldclient = tmp

			tmp = serverCount
			if serverCount < oldserver {
				serverCount += (math.MaxUint64 - oldserver)
			} else {
				serverCount -= oldserver
			}
			oldserver = tmp

			base := B
			danwei := ""
			switch *uint {
			case "k":
				base = K
				danwei = "K"
			case "m":
				base = M
				danwei = "M"
			}

			sspeed, cspeed := float64(serverCount)/float64(base), float64(clientCount)/float64(base)
			fmt.Printf("\r client %v speed: %9.2f %vbit/s, server %v speed: %9.2f %vbit/s         ", oldclient, cspeed*BITS, danwei, oldserver, sspeed*BITS, danwei)
		}
	}
}
