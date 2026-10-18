package main

import "fmt"

func main() {
	a := "foo" + "bar"
	fmt.Println(a)
	b := "hello, "
	c := b + "world"
	fmt.Println(c)
	fmt.Println("a" + "b" + "c")
	d := a + "!"
	fmt.Println(d)
	fmt.Println(len(d))
}
