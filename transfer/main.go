package main

import (
	"flag"
	"log"
	"net"
)

var (
	listen = flag.String("l", ":10556", "listen")
	dst    = flag.String("d", "192.168.0.103:10556", "destination address")
	// src        = flag.String("s", "192.168.0.109:10555", "source address")
	network    = flag.String("net", "udp", "network")
	packetsize = flag.Int("size", 2048, "packet size")
	print      = flag.Bool("v", false, "print")
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

	buf := make([]byte, *packetsize)
	var ltotal int
	for {
		var n int
		var err error
		n, src_addr, err = listen_conn.ReadFrom(buf)
		if *print {
			log.Println("recv from src", src_addr, n)
		}
		if n != 0 {
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
