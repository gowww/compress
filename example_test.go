package compress_test

import (
	"fmt"
	"net/http"

	"github.com/gowww/compress"
)

func Example() {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Hello")
	})

	http.ListenAndServe(":8080", compress.Handle(mux))
}

func ExampleHandle() {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Hello")
	})

	http.ListenAndServe(":8080", compress.Handle(mux))
}

func ExampleHandleFunc() {
	http.Handle("/", compress.HandleFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Hello")
	}))

	http.ListenAndServe(":8080", nil)
}
