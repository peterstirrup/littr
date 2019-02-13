package main

import (
	"flag"
	"fmt"
	"time"
)




var port = 8080

func main() {
	testFlag := flag.Bool("test_flag", false, "test flag")
	flag.Parse()

	func() {
		if *testFlag {
			fmt.Println("Hello World")
		}
	}()

	EvenDeeper()
	AnotherTestFun()

	//if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
	//	log.Printf("HTTP server terminated: %s\n", err)
	//}
}

func AnotherTestFun() {
	time.Sleep(2 * time.Second)
	EvenDeeper()
	EvennnnDeeper()
}

func EvenDeeper() {
	fmt.Println("Still do nothing")
	EvenEvenDeeper()
}

func EvenEvenDeeper() {
	fmt.Println("Doing nothing!")
}

func EvennnnDeeper() {
	fmt.Println("Still nothing")
}