package main

import "fmt"

func main() {
	s := make([]int, 0)
	s = append(s, 10)
	s = append(s, 20)
	fmt.Println(len(s))
	fmt.Println(s[0] + s[1])
	i := 0
	for i < len(s) {
		fmt.Println(s[i])
		i++
	}
	s = append(s, s[0]+s[1])
	fmt.Println(s[2])
	fmt.Println(len(s))
}
