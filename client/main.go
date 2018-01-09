package main

import (
	"flag"
	"log"
	"net/url"
	"time"
	"os"
	"os/signal"

	"github.com/gorilla/websocket"
	"sync"
)

var addr = flag.String("addr", "localhost:8080", "http service address")
var connections = flag.Int("connections", 5, "increased connection numbers per second")
var maxConns = flag.Int("maxConns", 1000000, "max numbers of connection to server")

var (
	mux sync.Mutex

	requestNums = 0
	keepAliveConns = 0
	failedConns = 0
)

func main() {
	flag.Parse()
	log.SetFlags(0)
	interrupt := make(chan os.Signal)
	signal.Notify(interrupt, os.Interrupt)

	u := url.URL{Scheme: "ws", Host: *addr, Path: "/ws"}

	go logMonitor()

	for ; *maxConns > 0; {
		for nums := *connections; nums > 0; nums-- {
			go CreateAndRunConnection(u)
			requestNums++;
		}
		*maxConns -= *connections
		time.Sleep(time.Second)
	}

	w := make(chan interface{})
	select {
	case <- w:
	case <- interrupt:
		return
	}
}

func CreateAndRunConnection(u url.URL) {
	logger := GetLogger()
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		logger.Fatal("dial", err)
		return
	}
	mux.Lock()
	keepAliveConns++
	mux.Unlock()
	defer c.Close()

	done := make(chan struct{})

	// loop read
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

				logger.Printf("read err:%v\n", err)
				return
			}
			logger.Println(message)
		}
	}()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	// loop write
	go func() {
		for {
			select {
			case t := <-ticker.C:
				err := c.WriteMessage(websocket.TextMessage, []byte(t.String()))
				if err != nil {
					log.Println("write:", err)
					return
				}
			case <-interrupt:
				log.Println("interrupt")
				// To cleanly close a connection, a client should send a close
				// frame and wait for the server to close the connection.
				err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				if err != nil {
					logger.Println("write close:", err)
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
	}()
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