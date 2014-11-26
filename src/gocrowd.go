package main

import (
    "io/ioutil"
    "log"
    "net/http"
    "os"
)

func handler(w http.ResponseWriter, r *http.Request) {
    uri := r.RequestURI
    if uri[len(uri)-1] == '/' {
        uri = uri + "index.html"
    }

    f := "htdocs"+uri

    file_content, err := ioutil.ReadFile(f)
    if err != nil {
        log.Print(err)
    }

    _, err = w.Write(file_content)
    if err != nil {
        log.Print(err)
    }
}

func main() {
    http.HandleFunc("/", handler)
    http.ListenAndServe(":"+os.Getenv("PORT"), nil)
}
