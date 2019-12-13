package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"net"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
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

	sigch := make(chan os.Signal)
	signal.Notify(sigch, syscall.SIGHUP)
	ticker := time.NewTicker(time.Second)
	var sold, cold uint64
	var ccnt, scnt uint64
	for {
		select {
		case <-ticker.C:
			cdiff, sdiff := calcDiff(&bytesFromClient, &ccnt, &cold), calcDiff(&bytesFromServer, &scnt, &sold)
			if cdiff == 0 && sdiff == 0 {
				continue
			}

			cspeed, sspeed := float64(cdiff)/basef, float64(sdiff)/basef
			cavg, savg := float64(cold)/float64(ccnt)/basef, float64(sold)/float64(scnt)/basef
			fmt.Printf("\r %9d\t%9.2f\t%9.2f\t%9d\t%9.2f\t%9.2f    ", cold, cavg, cspeed, sold, savg, sspeed)

		case sig := <-sigch:
			switch sig {
			case syscall.SIGHUP:
				atomic.StoreUint64(&bytesFromClient, 0)
				ccnt, cold = 0, 0
				atomic.StoreUint64(&bytesFromServer, 0)
				scnt, sold = 0, 0
			}
		}
	}
}

func calcDiff(addr, cnt, old *uint64) (cdiff uint64) {
	var tmp uint64
START:
	cdiff = atomic.LoadUint64(addr)
	if cdiff == *old {
		cdiff = 0
		return
	}

	if cdiff < *old { // 说明变量*addr发生了置位
		tmp = cdiff
		cdiff += (math.MaxUint64 - *old)
		if !atomic.CompareAndSwapUint64(&bytesFromClient, tmp, cdiff) {
			goto START
		}
		*cnt = 0 // 历史次数清零
		*old = cdiff
	} else {
		tmp = cdiff
		cdiff -= *old
		*old = tmp
	}
	*cnt++
	return
}
