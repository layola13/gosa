package main

import "fmt"

type Point struct {
	X int
	Y int
}

type Flag struct {
	On bool
	N  int
}

func main() {
	p := Point{X: 3, Y: 4}
	fmt.Println(p.X)
	fmt.Println(p.Y)
	p.X = 10
	fmt.Println(p.X + p.Y)
	q := Point{1, 2}
	fmt.Println(q.X + q.Y)
	f := Flag{On: true, N: 7}
	if f.On {
		fmt.Println(f.N)
	} else {
		fmt.Println(0)
	}
	f.N = f.N + 1
	fmt.Println(f.N)
}
