package main

import (
	"log"
	"os"
	"sync"
)

var (
	logger *log.Logger
	once sync.Once

)

func GetLogger() *log.Logger {
	once.Do(func() {
		logger = NewLogger()
	})
	return logger
}

func NewLogger() *log.Logger{
	logFile, err := os.OpenFile("./server.log", os.O_WRONLY | os.O_TRUNC |os.O_CREATE , 0777 )
	if err != nil {
		log.Println("open logfile error")
	}
	logger := log.New(logFile,"", log.Ldate | log.Ltime )
	return logger
}





