package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sync/atomic"
	"time"

	"github.com/DataDog/datadog-go/statsd"
)

var (
	in      = flag.String("in", "", "Incoming host:port")
	out     = flag.String("out", "", "Outgoing host:port")
	nostats = flag.Bool("no-stats", false, "Disables sending dogstatsd to outgoing host:port")
)

func printUsage() {
	fmt.Println("usage: udpfwd --in host:port --out host:port")
}

func main() {
	flag.Parse()
	if *in == "" || *out == "" {
		printUsage()
		os.Exit(0)
	}
	inaddr, err := net.ResolveUDPAddr("udp", *in)
	if err != nil {
		log.Fatal(err)
	}
	_, err = net.ResolveUnixAddr("unixgram", *out)
	if err != nil {
		log.Fatal(err)
	}
	inconn, err := net.ListenUDP("udp", inaddr)
	if err != nil {
		log.Fatal(err)
	}
	defer inconn.Close()
	outconn, err := net.Dial("unixgram", *out)
	if err != nil {
		log.Fatal(err)
	}
	defer outconn.Close()

	var (
		inbytes, outbytes   int64
		inerrors, outerrors int64
	)
	if !*nostats {
		stats, err := statsd.New("unix://" + *out)
		if err != nil {
			log.Printf("Statsd disabled: %v", err)
		} else {
			go func() {
				tick := time.NewTicker(10 * time.Second)
				defer tick.Stop()
				for {
					select {
					case <-tick.C:
						stats.Count("udpfwd.in_bytes", atomic.SwapInt64(&inbytes, 0), nil, 1)
						stats.Count("udpfwd.out_bytes", atomic.SwapInt64(&outbytes, 0), nil, 1)
						stats.Count("udpfwd.error", atomic.SwapInt64(&inerrors, 0), []string{"direction:in"}, 1)
						stats.Count("udpfwd.error", atomic.SwapInt64(&outerrors, 0), []string{"direction:out"}, 1)
					}
				}
			}()
		}
	}

	var buf [65535]byte
	for {
		nin, err := inconn.Read(buf[0:])
		if err != nil && err != io.EOF {
			atomic.AddInt64(&inerrors, 1)
			log.Printf("Error reading %d bytes: %v", nin, err)
		}
		atomic.AddInt64(&inbytes, int64(nin))
		nout, err := outconn.Write(buf[:nin])
		if err != nil {
			atomic.AddInt64(&outerrors, 1)
			log.Printf("Error writing %d bytes: %v", nout, err)
		}
		atomic.AddInt64(&outbytes, int64(nout))
	}
}
