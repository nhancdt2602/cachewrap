package main

import (
	"fmt"
	"runtime"
)

func test() {
	_, _, line, _ := runtime.Caller(1)
	fmt.Println("line:", line)
}


func main() {
//line main.go:100:1
	test()
}
