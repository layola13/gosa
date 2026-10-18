package main

import "fmt"

func main() {
	m := make(map[int]int)
	m[1] = 10
	m[2] = 20
	fmt.Println(m[1])
	fmt.Println(m[2])
	fmt.Println(len(m))
	m[1] = m[1] + m[2]
	fmt.Println(m[1])
	delete(m, 1)
	fmt.Println(len(m))
	fmt.Println(m[2])
}
