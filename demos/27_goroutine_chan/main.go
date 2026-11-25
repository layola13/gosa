package main

import "fmt"

func producer(out chan int, done chan bool) {
	var i int = 1
	for i <= 100 {
		out <- i * i
		i = i + 1
	}
	done <- true
}

func main() {
	ch := make(chan int)
	done := make(chan bool)
	go producer(ch, done)
	var total int = 0
	remaining := true
	for remaining {
		select {
		case v := <-ch:
			total = total + v
		case <-done:
			remaining = false
		}
	}
	fmt.Println(total)
}
