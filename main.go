package main

import (
	"flag"
	"net"

	"github.com/golang/glog"
)

var (
	addr                           string
	bufferSize, thread, retryTimes int
)

func init() {
	flag.StringVar(&addr, "l", ":8081", "addr to listen")
	flag.IntVar(&bufferSize, "b", 2*(1024<<10), "<bufferSize> per range")
	flag.IntVar(&thread, "t", 3, "concurrent range")
	flag.IntVar(&retryTimes, "r", 3, "retry times")
	flag.Set("logtostderr", "true")
	flag.Parse()
}

func main() {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		glog.Fatalln(err)
	}
	glog.Infof("ListenAndServe on %s", ln.Addr().String())
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			glog.Errorln(err)
			continue
		}
		go handleConn(conn)
	}
}
