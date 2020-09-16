package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"

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
	outaddr, err := net.ResolveUDPAddr("udp", *out)
	if err != nil {
		log.Fatal(err)
	}
	inconn, err := net.ListenUDP("udp", inaddr)
	if err != nil {
		log.Fatal(err)
	}
	defer inconn.Close()
	outconn, err := net.DialUDP("udp", nil, outaddr)
	if err != nil {
		log.Fatal(err)
	}
	defer outconn.Close()

	var stats statsdClient = &statsd.NoOpClient{}
	if !*nostats {
		stats, err = statsd.New(*out)
		if err != nil {
			log.Printf("Statsd disabled: %v", err)
			stats = &statsd.NoOpClient{}
		}
	}

	var buf [65535]byte
	for {
		nin, err := inconn.Read(buf[0:])
		if err != nil && err != io.EOF {
			stats.Count("udpfwd.error", 1, []string{"direction:in"}, 1)
			log.Printf("Error reading %d bytes: %v", nin, err)
		}
		stats.Count("udpfwd.in_bytes", int64(nin), nil, 1)
		nout, err := outconn.Write(buf[:nin])
		if err != nil {
			stats.Count("udpfwd.error", 1, []string{"direction:out"}, 1)
			log.Printf("Error writing %d bytes: %v", nout, err)
		}
		stats.Count("udpfwd.out_bytes", int64(nout), nil, 1)
	}
}

type statsdClient interface {
	Count(name string, value int64, tags []string, rate float64) error
}
