package common

import (
	"errors"
	"fmt"
	"net"
	"time"
)

func Dial(iface, host string, port int, timeout time.Duration, udp bool) (conn net.Conn, err error) {
	var network = "tcp"
	if udp {
		network = "udp"
	}

	if iface == "" {
		conn, err = net.DialTimeout(network, fmt.Sprintf("%v:%v", host, port), timeout)
	} else {
		switch network {
		case "tcp":
			var laddr, raddr *net.TCPAddr
			laddr, err = net.ResolveTCPAddr(network, iface+":0")
			if err != nil {
				return
			}
			raddr, err = net.ResolveTCPAddr(network, fmt.Sprintf("%v:%v", host, port))
			if err != nil {
				return
			}

			conn, err = net.DialTCP(network, laddr, raddr)

		case "udp":
			var laddr, raddr *net.UDPAddr
			laddr, err = net.ResolveUDPAddr(network, iface+":0")
			if err != nil {
				return
			}
			raddr, err = net.ResolveUDPAddr(network, fmt.Sprintf("%v:%v", host, port))
			if err != nil {
				return
			}

			conn, err = net.DialUDP(network, laddr, raddr)
		default:
			err = errors.New(network + " is not supported")
		}
	}

	if err != nil {
		fmt.Println("dial error:", err)
		return
	}
	return
}
