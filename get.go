package main

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"strconv"
	"strings"
	"sync"
	"time"
	"sync/atomic"

	"github.com/golang/glog"
)

func handleGet(conn net.Conn, req *http.Request) {
	req.Header.Del("Proxy-Connection")
	req.Header.Set("Connection", "Keep-Alive")

	resp, err := getResponseTimes(req)
	if err != nil {
		glog.Infoln(err)
		return
	}
	resp.Body.Close()

	contentLength := resp.ContentLength
	if contentLength <= int64(bufferSize) {
		handleNormalGet(conn, req)
	} else {
		handleRangeGet(conn, req, resp, contentLength)
	}
}

func handleNormalGet(conn net.Conn, req *http.Request) {
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

func handleRangeGet(conn net.Conn, req *http.Request, resp *http.Response, contentLength int64) {

	dump, err := httputil.DumpResponse(resp, false)
	if err != nil {
		glog.Infoln(err)
		return
	}
	conn.Write(dump)

	t := int(contentLength / int64(bufferSize))
	if (contentLength % int64(bufferSize)) != 0 {
		t += 1
	}

	ret := make([][]byte, t)
	var connState uint32 = 1
	cond := sync.NewCond(&sync.Mutex{})

	go doRangeQueue(ret, req, contentLength, t, cond, &connState, conn)

	for i := 1; i < t; i++ {
		cond.L.Lock()
		for ret[i] == nil {
			cond.Wait()

			if atomic.LoadUint32(&connState) == 0 {
				cond.L.Unlock()
				return
			}
		}
		cond.L.Unlock()

		_, err := conn.Write(ret[i])
		if err != nil {
			atomic.StoreUint32(&connState, 0)
			glog.Infoln(err)
			return
		}
		ret[i] = nil
	}
}

func fetchRange2ByteTimes(fetchReq *http.Request, fetchLength int64, connStatePtr *uint32) (ret []byte, err error) {

	for i := 0; i < retryTimes; i++ {
		if atomic.LoadUint32(connStatePtr) == 0 {
			return nil, fmt.Errorf("Conn State changed.\n")
		}

		ret, err = fetchRange2Byte(fetchReq, fetchLength, connStatePtr)
		if err != nil {
			continue
		}
		return
	}
	return
}

func fetchRange2Byte(fetchReq *http.Request, fetchLength int64, connStatePtr *uint32) ([]byte, error) {

	if atomic.LoadUint32(connStatePtr) == 0 {
		return nil, fmt.Errorf("Connect state changed.\n")
	}

	c, err := dialTimeoutTimes("tcp", fetchReq.Host, time.Second*30, retryTimes)
	if err != nil {
		return nil, err
	}
	defer c.Close()

	fetchReq.Write(c)

	var ret []byte
	buf := make([]byte, 32*1024)
	index := -1

	// get CR/LF index
	for {
		if atomic.LoadUint32(connStatePtr) == 0 {
			return nil, fmt.Errorf("Connect state changed.\n")
		}
		c.SetReadDeadline(time.Now().Add(time.Second * 30))
		n, err := c.Read(buf)
		if n > 0 {
			ret = append(ret, buf[:n]...)

			index = bytes.Index(ret, []byte{13, 10, 13, 10})
			if index != -1 {
				break
			}
		}

		if er, match := err.(net.Error); match && er.Timeout() {
			return nil, er
		}

		if err == io.EOF {
			if index == -1 {
				return nil, fmt.Errorf("index == -1.\n")
			}
			break
		}

		if err != nil {
			return nil, err
		}
	}

	total := int64(index+1) + 3 + fetchLength
	for {
		if atomic.LoadUint32(connStatePtr) == 0 {
			return nil, fmt.Errorf("Connect state changed.\n")
		}

		c.SetReadDeadline(time.Now().Add(time.Second * 30))
		n, err := c.Read(buf)
		if n > 0 {
			ret = append(ret, buf[:n]...)
			if int64(len(ret)) >= total {
				break
			}
		}

		if er, match := err.(net.Error); match && er.Timeout() {
			return nil, er
		}

		if err == io.EOF {
			if int64(len(ret)) < total {
				return nil, fmt.Errorf("len(ret)) < total.\n")
			}
			break
		}

		if err != nil {
			return nil, err
		}
	}

	return ret[index+4:], nil
}

func fetchRange2ConnTimes(fetchReq *http.Request, conn net.Conn, fetchLength int64) (err error) {

	for i := 0; i < retryTimes; i++ {
		err = fetchRange2Conn(fetchReq, conn, fetchLength)
		if err != nil {
			if er, match := err.(net.Error); match && er.Timeout() {
				continue
			}
			break
		}
		return
	}
	return
}

func fetchRange2Conn(fetchReq *http.Request, conn net.Conn, fetchLength int64) error {

	c, err := dialTimeoutTimes("tcp", fetchReq.Host, time.Second*30, retryTimes)
	if err != nil {
		return err
	}
	defer c.Close()

	fetchReq.Write(c)

	var ret []byte
	buf := make([]byte, 32*1024)
	index := -1

	// get CR/LF index
	for {
		c.SetReadDeadline(time.Now().Add(time.Second * 30))
		n, err := c.Read(buf)
		if n > 0 {
			ret = append(ret, buf[:n]...)

			index = bytes.Index(ret, []byte{13, 10, 13, 10})
			if index != -1 {
				break
			}
		}

		if er, match := err.(net.Error); match && er.Timeout() {
			return er
		}

		if err == io.EOF {
			if index == -1 {
				return fmt.Errorf("index == -1.\n")
			}
			break
		}

		if err != nil {
			return err
		}
	}

	conn.Write(ret[index+4:])

	total := int64(index+1) + 3 + fetchLength
	for {
		n, err := c.Read(buf)
		if n > 0 {
			conn.Write(buf[:n])

			ret = append(ret, buf[:n]...)
			if int64(len(ret)) >= total {
				break
			}
		}

		if err == io.EOF {
			if int64(len(ret)) < total {
				return fmt.Errorf("len(ret)) < total.\n")
			}
			break
		}

		if err != nil {
			return err
		}
	}

	return nil
}

func doRangeQueue(ret [][]byte, req *http.Request, contentLength int64, t int, cond *sync.Cond, connStatePtr *uint32, conn net.Conn) {

	rawBegin := func() (rawBegin int64) {
		if s := req.Header.Get("Range"); s != "" {
			ss := strings.Split(s, "-")
			rawBegin, _ = strconv.ParseInt(ss[0], 10, 0)
		} else {
			rawBegin = 0
		}
		return
	}()

	if err := doTheFirstRange(rawBegin, req, conn, connStatePtr, cond); err != nil {
		glog.Infoln(err)
		return
	}

	if err := doTheRestRange(rawBegin, req, connStatePtr, cond, ret, t, contentLength); err != nil {
		glog.Infoln(err)
		return
	}
}

func doTheFirstRange(rawBegin int64, req *http.Request, conn net.Conn, connStatePtr *uint32, cond *sync.Cond) error {
	fetchReq, fetchLength, err := func() (fetchReq *http.Request, fetchLength int64, err error) {
		header := fmt.Sprintf("%d-%d", rawBegin, rawBegin+int64(bufferSize-1))

		fetchReq, err = http.NewRequest("GET", req.URL.String(), nil)
		if err != nil {
			return
		}
		fetchReq.Header.Set("Range", "bytes="+header)
		fetchLength = int64(bufferSize)

		return
	}()
	if err != nil {
		return err
	}

	err = fetchRange2ConnTimes(fetchReq, conn, fetchLength)
	if err != nil {
		atomic.StoreUint32(connStatePtr, 0)
		cond.Signal()

		return err
	}

	cond.Signal()
	return nil
}

func doTheRestRange(rawBegin int64, req *http.Request, connStatePtr *uint32, cond *sync.Cond, ret [][]byte, t int, contentLength int64) error {
	thread := make(chan bool, thread)
	for i := 1; i < t; i++ {
		if atomic.LoadUint32(connStatePtr) == 0 {
			return fmt.Errorf("Connect state changed.\n")
		}

		fetchReq, fetchLength, err := getFetchArgs(i, t, rawBegin, contentLength, req)
		if err != nil {
			return err
		}

		thread <- true
		go func(i int) {
			subRet, err := fetchRange2ByteTimes(fetchReq, fetchLength, connStatePtr)
			if err != nil {
				atomic.StoreUint32(connStatePtr, 0)
				cond.Signal()

				return
			}
			<-thread
			_ = append(ret[:i], subRet)
			cond.Signal()
		}(i)
	}
	return nil
}

func getFetchArgs(i, t int, rawBegin, contentLength int64, req *http.Request) (fetchReq *http.Request, fetchLength int64, err error) {
	if i != t-1 {
		header := fmt.Sprintf("%d-%d", rawBegin+int64(bufferSize*i), rawBegin+int64(bufferSize*i+(bufferSize-1)))

		fetchReq, err = http.NewRequest("GET", req.URL.String(), nil)
		if err != nil {
			return
		}
		fetchReq.Header.Set("Range", "bytes="+header)
		fetchLength = int64(bufferSize)
	} else {
		header := fmt.Sprintf("%d-", rawBegin+int64(bufferSize*i))

		fetchReq, err = http.NewRequest("GET", req.URL.String(), nil)
		if err != nil {
			return
		}
		fetchReq.Header.Set("Range", "bytes="+header)
		fetchLength = contentLength - int64(bufferSize*i)
	}

	return
}
