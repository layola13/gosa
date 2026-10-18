package main

import "fmt"

func main() {
	s := "hello"
	fmt.Println(s)
	fmt.Println(len(s))
	t := s
	fmt.Println(t)
	s = "world!"
	fmt.Println(s)
	fmt.Println(len(s))
	fmt.Println(t)
}
