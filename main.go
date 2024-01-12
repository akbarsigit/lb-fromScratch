package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// make increment value with iota, attempts = 0, retry = 1
// keep track of the http request
const ( 
	Attempts int = iota
	Retry
)


type Backend struct {
	URL   *url.URL
	Alive bool
	mux   sync.RWMutex
	ReverseProxy *httputil.ReverseProxy
}

// keep track of the backend server
type ServerPool struct {
	backends []*Backend
	current uint64 // keep track of the index
}

func GetRetryFromContext(r *http.Request) int {
	fmt.Println(r.Context().Value(Retry))

	if retry, ok := r.Context().Value(Retry).(int); ok {
		return retry
	}
	return 0
}


func (b *Backend) SetAlive(alive bool) {
	b.mux.Lock()
	b.Alive = alive
	b.mux.Unlock()
}

func (s *ServerPool) MarkBackendStatus(backendUrl *url.URL, alive bool) {
	for _, b := range s.backends {
		if b.URL.String() == backendUrl.String() {
			b.SetAlive(alive)
			break
		}
	}
}

func GetAttemptsFromContext(r *http.Request) int {
	if attemps, ok := r.Context().Value(Attempts).(int); ok {
		return attemps
	}
	return 1
}

func (s *ServerPool) NextIndex() int {
	return int(atomic.AddUint64(&s.current, uint64(1)) % uint64(len(s.backends)))
}

// get the next active peer to connect
func (s *ServerPool) GetNextPeer() *Backend {
	// Find the alive backend in the pool
	next := s.NextIndex()
}

// Load balancing
func lb(w http.ResponseWriter, r *http.Request) {
	attempts := GetAttemptsFromContext(r)
	if attempts > 3 {
		log.Printf("%s(%s) Max attemps reached, terminating\n", r.RemoteAddr, r.URL.Path)
		http.Error(w, "servie not available", http.StatusServiceUnavailable)
		return
	}

	peer := serverPool.GetNextPeer()
	if peer != nil {
		peer.ReverseProxy.ServeHTTP(w, r)
	}

}

var serverPool ServerPool

func main() {
	var serverList string
	var port int

	// cli argument, -backend=server1,server2 .... -port=8080
	// seperate using comma, dont use space
	flag.StringVar(&serverList, "backend", "", "Load balancer backend, separate with commas.")
	flag.IntVar(&port, "port", 3030, "Port to serve")

	flag.Parse()

	if len(serverList) == 0 {
		log.Fatal("Please provide one or more backends to load balance")
	}

	// parse servers
	tokens := strings.Split(serverList, ",")
	for _, tok := range tokens {
		serverUrl, err := url.Parse(tok)
		if err != nil {
			log.Fatal(err)
		}
		// log.Printf("Configured server: %s\n", serverUrl)
		
		// all request will be passed to the serverUrl 
		proxy := httputil.NewSingleHostReverseProxy(serverUrl)
		proxy.ErrorHandler = func(writer http.ResponseWriter, request *http.Request, e error){
			log.Printf("[%s] %s\n", serverUrl.Host, e.Error())
			retries := GetRetryFromContext(request)

			// we try 3 times for a request to reach server
			if retries < 3 {
				select {
				case <- time.After(10 * time.Millisecond):
					ctx := context.WithValue(request.Context(), Retry, retries+1)
					proxy.ServeHTTP(writer, request.WithContext(ctx))
				}
				return
			}

			// after 3 retreis, mark it as backend down
			serverPool.MarkBackendStatus(serverUrl, false)


			// if the same request routing for few attempts with different backends, increase the count
			attempts := GetAttemptsFromContext(request)
			log.Printf("%s(%s) Attempting retry %d\n", request.RemoteAddr, request.URL.Path, attempts)
			ctx := context.WithValue(request.Context(), Attempts, attempts+1)
			lb(writer, request.WithContext(ctx))
		}
		
		


		// fmt.Println(proxy)

	}
}