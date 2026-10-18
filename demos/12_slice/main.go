package main

import "fmt"

func main() {
	s := make([]int, 5)
	fmt.Println(len(s))
	s[0] = 10
	s[1] = 20
	s[4] = s[0] + s[1]
	fmt.Println(s[0])
	fmt.Println(s[4])
	sum := 0
	i := 0
	for i < len(s) {
		sum = sum + s[i]
		i++
	}
	fmt.Println(sum)
}
