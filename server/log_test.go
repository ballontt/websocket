package main

import "testing"

func TestGetLogger(t *testing.T) {
	num := 5
	for ;num > 0; {
		go printLog(num)
		num--
	}
}

func printLog(index int) {
	logger := GetLogger()
	logger.Println(index)
}


