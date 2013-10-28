package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"time"
)

var num_conn uint32 = 1014

const svr_address = "shoutcast.server.com:80"
const tick_interval = 500
const buffer_size = 4 * 1024
const print_progress = false

func create_getter(uri string,
	bufsize int,
	quit chan uint16,
	death chan uint16) func(uint32) {
	return func(id uint32) {
		defer func() { death <- 1 }()
		if print_progress {
			fmt.Println()
		}
		fmt.Println("starting go routine", id)
		var cnt uint32 = 0
		buff := make([]byte, bufsize)
		conn, _ := net.Dial("tcp", uri)
		defer conn.Close()
		fmt.Fprint(conn, "GET / HTTP/1.0\r\n\r\n")
		for {
			select {
			case <-quit:
				if print_progress {
					fmt.Println()
				}
				fmt.Println("terminating go routine", id)
				return
			default:
				if n, err := conn.Read(buff); n == 0 {
					if print_progress {
						fmt.Println()
					}
					fmt.Println("Reading connection", id, "failed:", err)
					return
				}
				if cnt++; print_progress && cnt%num_conn == 0 {
					fmt.Print(".")
				}
			}
		}
	}
}

func main() {
	quit := make(chan uint16)
	death := make(chan uint16, 8)

	get := create_getter(svr_address, buffer_size, quit, death)

	s := make(chan os.Signal, 2)
	signal.Notify(s)
	var sig os.Signal

	ticker := time.NewTicker(tick_interval * time.Millisecond)
	var born, dead uint32
M:
	for born < num_conn {
		select {
		case <-ticker.C:
			born++
			go get(born)
		case sig = <-s:
			break M
		}
	}
	ticker.Stop()

	for sig == nil && dead < born {
		select {
		case sig = <-s:
		case <-death:
			dead++
		}
	}
	signal.Stop(s)

	if print_progress {
		fmt.Println()
	}
	fmt.Println("Terminating", born-dead, "go routine(s)..")

	for dead < born {
		select {
		case quit <- uint16(1):
		case <-death:
			dead++
			fmt.Println("Alive:", born-dead, "go routine(s).")
		}
	}

	if print_progress {
		fmt.Println()
	}
	fmt.Println("Got signal:", sig)
}
