package main

import (
    "fmt"
    "log"
    "net/http"
    "os"
)

func handler(w http.ResponseWriter, r *http.Request) {
    f := "htdocs"+r.RequestURI

    file_content, err := ioutil.ReadFile(r.RequestURI)
    if err != nil {
        log.Error(err)
    }

    err := w.Write(file_content)
    if err != nil {
        log.Error(err)
    }
}

func main() {
    http.HandleFunc("/", handler)
    http.ListenAndServe(":"+os.Getenv("PORT"), nil)
}
