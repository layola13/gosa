package main

import "fmt"

func main() {
	x := 2
	switch x {
	case 1:
		fmt.Println(10)
	case 2, 3:
		fmt.Println(20)
	default:
		fmt.Println(30)
	}
	y := 9
	switch y {
	case 1:
		fmt.Println(10)
	default:
		fmt.Println(99)
	}
	i := 0
	for i < 5 {
		i++
		switch i {
		case 2:
			continue
		case 4:
			break
		}
		fmt.Println(i)
	}
}
