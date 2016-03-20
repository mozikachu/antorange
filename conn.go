package main

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/golang/glog"
)

func handleConn(conn net.Conn) {
	defer conn.Close()

	req, err := http.ReadRequest(bufio.NewReader(conn))
	if err != nil {
		glog.Infoln(err)
		return
	}

	switch req.Method {
	case http.MethodConnect:
		b := []byte("HTTP/1.1 200 Connection established\r\n" + "Proxy-Agent: antorange" + "\r\n\r\n") // todo
		c, err := dialTimeoutTimes("tcp", req.Host, time.Second*30, retryTimes)
		if err != nil {
			glog.Infoln(err)
			return
		}
		defer c.Close()
		if _, err := conn.Write(b); err != nil {
			glog.Infoln(err)
			return
		}
		Transport(conn, c)
	case http.MethodGet:
		handleGet(conn, req)
	default:
		req.Header.Del("Proxy-Connection")
		req.Header.Set("Connection", "Keep-Alive")

		c, err := dialTimeoutTimes("tcp", req.Host, time.Second*30, retryTimes)
		if err != nil {
			glog.Infoln(err)
			return
		}
		defer c.Close()

		if err = req.Write(c); err != nil {
			glog.Infoln(err)
			return
		}
		Transport(conn, c)
	}
	return
}

func Transport(conn, c net.Conn) (err error) {
	wChan := make(chan error, 1)
	rChan := make(chan error, 1)

	go Pipe(c, conn, wChan)
	go Pipe(conn, c, rChan)

	select {
	case err = <-wChan:
	case err = <-rChan:
	}

	return
}

func Pipe(dst io.Writer, src io.Reader, ch chan<- error) {
	_, err := Copy(dst, src)
	ch <- err
}

// based on io.Copy
func Copy(dst io.Writer, src io.Reader) (written int64, err error) {
	buf := make([]byte, 32*1024)

	for {
		nr, er := src.Read(buf)
		//log.Println("cp r", nr, er)
		if nr > 0 {
			nw, ew := dst.Write(buf[:nr])
			//log.Println("cp w", nw, ew)
			if nw > 0 {
				written += int64(nw)
			}
			if ew != nil {
				err = ew
				break
			}
			/*
				if nr != nw {
					err = io.ErrShortWrite
					break
				}
			*/
		}
		if er == io.EOF {
			break
		}
		if er != nil {
			err = er
			break
		}
	}
	return
}
