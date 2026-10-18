package main

import "fmt"

func main() {
	x := 7
	if x > 5 {
		fmt.Println(1)
	} else {
		fmt.Println(0)
	}
	if x < 5 {
		fmt.Println(1)
	} else {
		fmt.Println(0)
	}
	if x == 7 {
		fmt.Println(x)
	} else {
		fmt.Println(-1)
	}
	if x != 7 {
		fmt.Println(0)
	} else if x >= 7 {
		fmt.Println(2)
	} else {
		fmt.Println(3)
	}
}
