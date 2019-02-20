package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
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

	// Set handler
	http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Printf("Error reading body: %v", err)
			http.Error(w, "can't read body", http.StatusBadRequest)
			return
		}

		w.Write(body)

		if string(body) == "quit" {
			w.Write([]byte("quitting"))
			os.Exit(0)
		}
	})

	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
		fmt.Printf("HTTP server terminated: %s\n", err)
	}
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