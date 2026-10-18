package main

import "fmt"

func main() {
	i := 0
	for i < 10 {
		i++
		if i == 3 {
			continue
		}
		if i == 8 {
			break
		}
		fmt.Println(i)
	}
	sum := 0
	j := 0
	for j < 5 {
		j++
		if j%2 == 0 {
			continue
		}
		sum = sum + j
	}
	fmt.Println(sum)
}
