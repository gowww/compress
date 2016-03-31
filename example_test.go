package compress

import (
	"fmt"
	"net/http"
)

func Example() {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Response is gzipped when content is long enough.")
	})

	http.ListenAndServe(":8080", Handle(mux))
}

func ExampleHandle() {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Response is gzipped when content is long enough.")
	})

	http.ListenAndServe(":8080", Handle(mux))
}

func ExampleHandleFunc() {
	http.Handle("/", HandleFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Response is gzipped when content is long enough.")
	}))

	http.ListenAndServe(":8080", nil)
}
