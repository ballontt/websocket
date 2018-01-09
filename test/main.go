package main

import (
	"flag"
	"log"
	"net/url"
	"os"
	"os/signal"
	"time"
	"sync"

	"github.com/gorilla/websocket"
)

var addr = flag.String("addr", "localhost:8080", "http service address")
var connections = flag.Int("conn", 5, "increased connection numbers per second")
var maxConns = flag.Int("maxconn", 1000000, "max numbers of connection to server")

var (
	mux sync.Mutex

	requestNums = 0
	keepAliveConns = 0
	failedConns = 0
)

func main() {

	go logMonitor()

	for ; *maxConns > 0; {
		for nums := *connections; nums > 0; nums-- {
			go connWebSocket(nums)
			requestNums++;
		}
		*maxConns -= *connections
		time.Sleep(time.Second)
	}

	time.Sleep(time.Minute * 60)
}

func connWebSocket(num int) {
	flag.Parse()
	log.SetFlags(0)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u := url.URL{Scheme: "ws", Host: *addr, Path: "/ws"}
	log.Printf("connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		mux.Lock()
		keepAliveConns--
		failedConns++
		mux.Unlock()
		log.Fatal("dial:", err)
		return
	}
	keepAliveConns++

	defer c.Close()

	done := make(chan struct{})

	go func() {
		defer c.Close()
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				mux.Lock()
				keepAliveConns--
				failedConns++
				mux.Unlock()

				log.Println("read:", err)
				return
			}
			log.Printf("goroution %d, recv: %s", num, message)
		}
	}()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case  t := <-ticker.C:
			err := c.WriteMessage(websocket.TextMessage, []byte(t.String()))
			if err != nil {
				log.Println("write:", err)
				mux.Lock()
				keepAliveConns--
				failedConns++
				mux.Unlock()
				return
			}
		case <-interrupt:
			log.Println("interrupt")
			// To cleanly close a connection, a client should send a close
			// frame and wait for the server to close the connection.
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close:", err)
				return
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			mux.Lock()
			keepAliveConns--
			failedConns++
			mux.Unlock()

			c.Close()
			return
		}
	}
}

func logMonitor() {
	logFile, err := os.OpenFile("./websocket.log", os.O_WRONLY | os.O_TRUNC | os.O_CREATE, 0666)
	if err != nil {
		log.Printf("open logFile err: %v\n", err.Error())
	}
	logger := log.New(logFile, "Info", log.Ldate | log.Ltime )

	ticker := time.NewTicker(time.Second * 5)
	for {
		select {
		case <- ticker.C:
			logger.Printf("Requests: %d, KeepAlives: %d, Faileds: %d", requestNums, keepAliveConns, failedConns)
		}
	}
}
