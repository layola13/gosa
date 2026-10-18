package main

import "fmt"

func isPos(x int) bool {
	return x > 0
}

func main() {
	a := true
	b := false
	fmt.Println(a)
	fmt.Println(!a)
	fmt.Println(a && true)
	fmt.Println(a || b)
	x := 5
	fmt.Println(isPos(x))
	fmt.Println(isPos(-3))
	if a && isPos(x) {
		fmt.Println(1)
	} else {
		fmt.Println(0)
	}
	for b {
		fmt.Println(99)
		break
	}
}
