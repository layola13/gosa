package main

import "fmt"

const TopN = 40
const TopFlag = true
const TopMsg = "top"

func main() {
	const LocalM = 2
	const LocalT = "hi"
	fmt.Println(TopN + LocalM)
	if TopFlag {
		fmt.Println(TopMsg)
	} else {
		fmt.Println("no")
	}
	fmt.Println(LocalT)
	sum := TopN
	i := 0
	for i < LocalM {
		sum = sum + i
		i++
	}
	fmt.Println(sum)
}
