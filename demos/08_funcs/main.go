package main

import "fmt"

func add(a int, b int) int {
	return a + b
}

func printTwice(x int) {
	fmt.Println(x)
	fmt.Println(x + 1)
}

func main() {
	y := add(40, 2)
	fmt.Println(y)
	fmt.Println(add(1, 2))
	printTwice(7)
}
