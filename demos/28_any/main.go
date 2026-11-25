package main

import "fmt"

func val(x int) any {
	if x == 0 {
		return true
	}
	return x * 2
}

func main() {
	m := make(map[string]int)
	m["a"] = 10
	m["b"] = 32
	xs := []int{1, 2, 3}
	xs = append(xs, 4)
	var total int = 0
	total = total + m["a"] + m["b"]
	for _, v := range xs {
		total = total + v
	}
	switch v := val(21).(type) {
	case int:
		total = total + v
	case bool:
		total = total + 1000
	}
	fmt.Println(total)
}
