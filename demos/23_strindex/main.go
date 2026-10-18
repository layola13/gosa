package main

import "fmt"

func main() {
	s := "hello"
	fmt.Println(s[0])
	fmt.Println(s[1])
	fmt.Println(s[4])
	fmt.Println(len(s))
	i := 0
	for i < len(s) {
		fmt.Println(s[i])
		i++
	}
}
