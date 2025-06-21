package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type RoundRobin struct {
	backends []string
	index    int
	lock     sync.Mutex
}

func (r *RoundRobin) Initialise(backends []string) error {
	if len(backends) == 0 {
		return fmt.Errorf("empty backends list")
	}
	r.backends = backends
	r.index = 0
	r.lock = sync.Mutex{}
	return nil
}

func (r *RoundRobin) Next() string {
	r.lock.Lock()
	backend := r.backends[r.index]
	r.index = (r.index + 1) % len(r.backends)
	r.lock.Unlock()
	return backend
}

type LoadBalancer struct {
	BackendsJSON string
	Backends     []string
	// Algo string
	Server   *http.Server
	Client   *http.Client
	Strategy *RoundRobin
}

func (l *LoadBalancer) Initialise(port string, jsonpath string) error {
	//read and parse the json path into backends

	handler := http.NewServeMux()
	handler.HandleFunc("/", l.Lbhandler)

	BACKENDS := []string{"localhost:3000", "localhost:4000"} // to be parsed from json

	l.Server = &http.Server{
		Addr:         ":" + port,
		IdleTimeout:  120 * time.Second,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		Handler:      handler,
	}

	l.Client = &http.Client{
		Timeout: 10 * time.Second,
	}

	l.Strategy = &RoundRobin{}
	err := l.Strategy.Initialise(BACKENDS)
	if err != nil {
		return err
	}

	return nil
}

func (l *LoadBalancer) Start() error {
	log.Println("starting loadbalancer ...")
	log.Println("started loadbalancer on ", l.Server.Addr)
	if err := l.Server.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (l *LoadBalancer) Stop(ctx context.Context) error {
	log.Println("stropping loadbalancer")
	if err := l.Server.Shutdown(ctx); err != nil {
		return err
	}
	log.Println("loadbalancer taken care of.")
	return nil
}

// sample function that creates request and calls it using client

// func somehandler(w http.ResponseWriter, r *http.Request) {
// 	newclient := &http.Client{}
// 	request, err := http.NewRequest(http.MethodGet, "http://localhost:4000/health", nil)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	response, err := newclient.Do(request)
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	defer response.Body.Close()

// 	if response.StatusCode == http.StatusOK {
// 		bodybytes, err := io.ReadAll(response.Body)
// 		if err != nil {
// 			log.Fatal(err)
// 		}
// 		w.WriteHeader(http.StatusOK)
// 		w.Write(bodybytes)
// 	} else {
// 		w.WriteHeader(http.StatusInternalServerError)
// 	}

// }

var hopHeaders = map[string]struct{}{
	"Connection":          {},
	"Proxy-Connection":    {},
	"Keep-Alive":          {},
	"Proxy-Authenticate":  {},
	"Proxy-Authorization": {},
	"TE":                  {},
	"Trailer":             {},
	"Transfer-Encoding":   {},
	"Upgrade":             {},
}

func (l *LoadBalancer) Lbhandler(w http.ResponseWriter, r *http.Request) {
	// loadbalancer is essentially a reverse proxy
	// determine the scheme to be used ( http or https )
	// determine the backend url and port to forward to ( lb algo )
	// copy the request url path and query params
	// copy the headers except those that should not be copied as per RFC7230
	// copy request body

	scheme := "http://"
	targethost := l.Strategy.Next()
	targetURL := scheme + targethost + r.URL.RequestURI()

	request, err := http.NewRequestWithContext(r.Context(), r.Method, targetURL, r.Body)
	if err != nil {
		log.Println("failed to create request")
		w.WriteHeader(http.StatusInternalServerError)
	}
	for k, v := range r.Header {
		if _, exists := hopHeaders[k]; exists {
			continue
		} else {
			for _, j := range v {
				request.Header.Add(k, j)
			}
		}
	}

	resp, err := l.Client.Do(request)
	if err != nil {
		log.Println("error during request execution : ", err)
	}

	for k, v := range resp.Header {
		for _, j := range v {
			w.Header().Add(k, j)
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)

}

func main() {

	var lb LoadBalancer
	err := lb.Initialise("4242", "somerandompath")
	if err != nil {
		log.Fatalf("error during initialisation : %v", err)
	}

	go func() {
		if err := lb.Start(); err != nil {
			log.Fatalf("error during startup : %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop //listen for activity on channel; blocking

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := lb.Stop(ctx); err != nil {
		log.Fatalf("error during shutdown : %v", err)
	}
}
