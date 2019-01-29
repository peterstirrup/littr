package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
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

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		AnotherTestFun()
		fmt.Println("In server handler")
	})

	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
		log.Printf("HTTP server terminated: %s\n", err)
	}
}

func AnotherTestFun() {
	time.Sleep(time.Second)
	EvenDeeper()
	EvennnnDeeper()
}

func EvenDeeper(){
	fmt.Println("Still do nothing")
	EvenEvenDeeper()
}

func EvenEvenDeeper() {
	fmt.Println("Doing nothing!")
}

func EvennnnDeeper() {
	fmt.Println("Still nothing")
}