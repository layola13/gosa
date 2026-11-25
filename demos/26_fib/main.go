package main

import "fmt"

func fib(n int) int {
	if n < 2 {
		return n
	}
	return fib(n-1) + fib(n-2)
}

func main() {
	var sum int = 0
	var i int = 0
	for i < 2000000 {
		sum = sum + i
		i = i + 1
	}
	fmt.Println(fib(28))
	fmt.Println(sum)
}
