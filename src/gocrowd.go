package main

import (
    "io/ioutil"
    "log"
    "net/http"
    "os"
)

func handler(w http.ResponseWriter, r *http.Request) {
    uri := r.RequestURI
    f := "htdocs"+uri

    fi, err := os.Stat(f)
    if err != nil {
        log.Print(err)
    }

    if fi.IsDir() {
        if f[len(f)-1] != '/' {
            f = f + "/"
        }
        f = f + "index.html"
    }

    fi, err = os.Stat(f)
    if err != nil {
        log.Print(err)
    }


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
