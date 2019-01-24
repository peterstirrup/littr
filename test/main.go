package main

import (
	"flag"
	"fmt"
)

func main() {
	testFlag := flag.Bool("test_flag", false, "test flag")
	flag.Parse()

	func() {
		if *testFlag {
			fmt.Println("Hello World")
		}
	}()

	AnotherTestFun()
}

func AnotherTestFun() {
	fmt.Println("Do nothing")
}