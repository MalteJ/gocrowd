package main

import (
    "bytes"
    "encoding/base64"
    "encoding/json"
    "io/ioutil"
    "log"
    "mime"
    "net/http"
    "net/url"
    "os"
    "strconv"
    "strings"
    "sync"
)

var max_cached_file_size, _ = strconv.ParseInt(os.Getenv("MAX_CACHED_FILE_SIZE"), 10, 64)
var max_cache_size, _ = strconv.ParseInt(os.Getenv("MAX_CACHE_SIZE"), 10, 64)
var cached_bytes = int64(0)

type Item struct {
    Data []byte
    Ttl int64
    Mimetype string
}

type Cache struct {
    items map[string]Item
    lock  sync.Mutex
}

func (c *Cache) Set(key string, value Item) {
    c.lock.Lock()
    c.items[key] = value
    c.lock.Unlock()
}

func (c *Cache) Get(key string) (Item, bool) {
    c.lock.Lock()
    value, ok := c.items[key]
    c.lock.Unlock()
    return value, ok
}

func NewCache() *Cache {
    return &Cache{
        items: map[string]Item{},
        lock: sync.Mutex{},
    }
}

var cache = NewCache()
var authCache = NewCache()

type CrowdAuthRequest struct {
    Value string `json:"value"`
}

func authenticate(username, password string) bool {
    if i, ok := authCache.Get(username); ok && string(i.Data) == password {
        log.Print("AuthCache Hit: "+username)
        return true
    }

    url := os.Getenv("CROWD_URL")+"/rest/usermanagement/1/authentication?username=" + username
    body_struct := &CrowdAuthRequest{Value:password}
    body, _ := json.Marshal(body_struct)

    req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
    req.SetBasicAuth(os.Getenv("CROWD_APP_NAME"), os.Getenv("CROWD_APP_PASSWORD"))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Accept", "application/json")

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        log.Print(err)
    }
    defer resp.Body.Close()

    if os.Getenv("DEBUG") == "TRUE" {
        resp_body_bytes, _ := ioutil.ReadAll(resp.Body)
        resp_body := string(resp_body_bytes)
        log.Print("Crowd Response Code: "+resp.Status)
        log.Print("Crowd Response Body: "+resp_body)
    }

    if resp.StatusCode == 200 {
        log.Printf("Adding User '%s' to AuthCache", username)
        authCache.Set(username, Item{Data: []byte(password)})
        return true
    } else {
        return false
    }
}

func handler(w http.ResponseWriter, r *http.Request) {
    // Check for HTTPS Connection
    if os.Getenv("FORCE_HTTPS") == "TRUE" && r.Header.Get("X-Forwarded-Proto") != "https" {
        https_url := url.URL{
            Scheme: "https",
            Host: r.Host,
            User: r.URL.User,
            Path: r.URL.Path,
            RawQuery: r.URL.RawQuery,
            Fragment: r.URL.Fragment,
        }
        w.Header().Set("Location", https_url.String())
        w.WriteHeader(301)
        return
    }

    // Check if User is authorized
    auth_header := r.Header.Get("Authorization")
    if strings.HasPrefix(auth_header, "Basic ") {  // Request has Basic Auth Header
        b64 := strings.Split(auth_header, " ")
        if len(b64) != 2 {
            w.WriteHeader(400)
            w.Write([]byte("<h1>Bad Request</h1>"))
            return
        }
        data, err := base64.StdEncoding.DecodeString(b64[1])
        if err != nil {
            log.Print("error decoding base64:", err)
            w.WriteHeader(500)
            w.Write([]byte("<h1>Internal Server Error</h1>"))
            return
        }
        spl := strings.SplitN(string(data), ":", 2)
        if len(spl) != 2 {
            w.WriteHeader(400)
            w.Write([]byte("<h1>Bad Request</h1>"))
            return
        }
        username := spl[0]
        password := spl[1]

        authorized := authenticate(username, password)
        if !authorized { // Send Login Request again
            w.Header().Set("WWW-Authenticate", `Basic realm="hybris Crowd Login"`)
            w.WriteHeader(401)
            w.Write([]byte("<h1>Unauthorized</h1><p>Please sign in with your Crowd credentials.</p>"))
            return            
        }
    } else {  // No Authentication Information provided
        w.Header().Set("WWW-Authenticate", `Basic realm="hybris Crowd Login"`)
        w.WriteHeader(401)
        w.Write([]byte("<h1>Unauthorized</h1><p>Please sign in with your Crowd credentials.</p>"))
        return
    }

    // Check if path is already in Cache
    path := r.URL.Path

    item, ok := cache.Get(path)
    if ok {  // Get Content from Cache
        log.Printf("ContentCache Hit: %s (Content-Type: %s)", path, item.Mimetype)
        w.Header().Set("Content-Type", item.Mimetype)
        w.Write(item.Data)
        return
    }

    // Load content from disk
    f := "htdocs"+path

    fi, err := os.Stat(f)
    if err != nil {
        if os.IsNotExist(err) {
            log.Print("404: "+path)
            w.WriteHeader(404)
            w.Write([]byte("<h1>Not Found</h1>The requested URL was not found on this server."))
            return
        }
        log.Print(err)
        return
    }

    if fi.IsDir() {
        if f[len(f)-1] != '/' {
            f = f + "/"
        }
        f = f + "index.html"
        fi, err = os.Stat(f)
        if os.IsNotExist(err) {
            log.Print("403: "+path)
            w.WriteHeader(403)
            w.Write([]byte("<h1>Forbidden</h1>You are not allowed to access this resource."))
            return
        } else if err != nil {
            log.Print(err)
        }
    }

    // Read File
    file_size := fi.Size()

    // save Content in Cache if sizes match and send content to client
    if file_size <= max_cached_file_size && cached_bytes + file_size <= max_cache_size {
        cached_bytes = cached_bytes + file_size

        file_content, err := ioutil.ReadFile(f)
        if err != nil {
            log.Print(err)
        }


        f_spl := strings.Split(f, ".")
        ext := "."+f_spl[len(f_spl)-1]
        mimetype := mime.TypeByExtension(ext)
        log.Printf("Caching %s with Content-Type: %s", f, mimetype)

        cache.Set(path, Item{Data: file_content, Mimetype: mimetype})

        w.Header().Set("Content-Type", mimetype)
        _, err = w.Write(file_content)
        if err != nil {
            log.Print(err)
        }
    } else {  // don't cache, stream directly from disk to client
        log.Print("Do not cache "+f)
        w.WriteHeader(418)
        w.Write([]byte("<h1>I'm a teapot</h1>"))
    }
}

func main() {
    http.HandleFunc("/", handler)
    http.ListenAndServe(":"+os.Getenv("PORT"), nil)
}
