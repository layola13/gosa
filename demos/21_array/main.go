package main

import "fmt"

func main() {
	var a [3]int
	a[0] = 7
	a[1] = 8
	fmt.Println(a[0] + a[1])
	fmt.Println(len(a))
	b := [2]int{10, 20}
	fmt.Println(b[0] + b[1])
	c := [3]int{1: 100, 2: 200}
	fmt.Println(c[0] + c[1] + c[2])
	sum := 0
	i := 0
	for i < len(a) {
		sum = sum + a[i]
		i++
	}
	fmt.Println(sum)
}
