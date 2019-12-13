package main

import (
	"flag"
	"fmt"
	"log"
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
	unit       = flag.String("unit", "k", "bits unit: m,k,b ")

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
	base := B
	danwei := ""
	switch *unit {
	case "k":
		base = K
		danwei = "K"
	case "m":
		base = M
		danwei = "M"
	}
	fmt.Printf("client bytes\tavg(%vbits/s)\treal(%vbits/s)\tserver bytes\tavg(%vbits/s)\treal(%vbits/s)\n", danwei, danwei, danwei, danwei)
	basef := float64(base / BITS)

	ticker := time.NewTicker(time.Second)
	var oldserver, oldclient, tmp uint64
	var ccnt, scnt uint64
	for {
		select {
		case <-ticker.C:
			serverCount, clientCount := atomic.LoadUint64(&bytesFromServer), atomic.LoadUint64(&bytesFromClient)
			if clientCount == oldclient && serverCount == oldserver {
				continue
			}

			ccnt++
			// if clientCount < oldclient {
			// clientCount += (math.MaxUint64 - oldclient)
			// ccnt = 1
			// oldclient = clientCount
			// } else {
			tmp = clientCount
			clientCount -= oldclient
			oldclient = tmp
			// }

			scnt++
			// if serverCount < oldserver {
			// 	serverCount += (math.MaxUint64 - oldserver)
			// 	scnt = 1
			// 	oldserver = serverCount
			// } else {
			tmp = serverCount
			serverCount -= oldserver
			oldserver = tmp
			// }

			cspeed, sspeed := float64(clientCount)/basef, float64(serverCount)/basef
			cavg, savg := float64(oldclient)/float64(ccnt)/basef, float64(oldserver)/float64(scnt)/basef
			fmt.Printf("\r %9d\t%9.2f\t%9.2f\t%9d\t%9.2f\t%9.2f       ", oldclient, cavg, cspeed, oldserver, savg, sspeed)
		}
	}
}
