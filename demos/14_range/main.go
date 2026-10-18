package main

import "fmt"

func main() {
	s := make([]int, 0)
	s = append(s, 10)
	s = append(s, 20)
	s = append(s, 30)
	sum := 0
	for i, v := range s {
		sum = sum + i + v
	}
	fmt.Println(sum)
	prod := 0
	for i := range s {
		prod = prod + i
		if i == 1 {
			break
		}
	}
	fmt.Println(prod)
	cnt := 0
	for _, v := range s {
		if v == 20 {
			continue
		}
		cnt = cnt + v
	}
	fmt.Println(cnt)
}
