package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	consts "github.com/nn-advith/loadbalancer/consts"
	strategy "github.com/nn-advith/loadbalancer/strategy"
	logger "github.com/nn-advith/loadbalancer/utils/logger"
	parser "github.com/nn-advith/loadbalancer/utils/parser"
)

type LoadBalancer struct {
	Server   *http.Server
	Client   *http.Client
	Strategy strategy.Strategy
}

func GetStrategy() (strategy.Strategy, error) {
	strategyval := os.Getenv("NBLB_STRATEGY")
	if _, exists := strategy.StrategyMap[strategyval]; !exists {
		return nil, fmt.Errorf("unknown loadbalancing strategy: %v", strategyval)
	} else {
		return strategy.StrategyMap[strategyval], nil
	}
}

func (l *LoadBalancer) Initialise(port string, jsonpath string) error {
	//read and parse the json path into backends
	// check if the json directory is present and whether the file is accessible.
	// if yes, try decode. if no values present; throw error

	handler := http.NewServeMux()
	handler.HandleFunc("/", l.Lbhandler)

	BACKENDS, err := parser.ParseBackendJSON()
	if err != nil {
		return err
	}

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

	lbstrategy, err := GetStrategy()
	if err != nil {
		return err
	}
	l.Strategy = lbstrategy
	err = l.Strategy.Initialise(BACKENDS)
	if err != nil {
		return err
	}

	return nil
}

func (l *LoadBalancer) Start() error {
	logger.L.Println("starting loadbalancer ...")
	logger.L.Println("started loadbalancer on ", l.Server.Addr)
	if err := l.Server.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (l *LoadBalancer) Stop(ctx context.Context) error {
	logger.L.Println("stropping loadbalancer")
	if err := l.Server.Shutdown(ctx); err != nil {
		return err
	}
	logger.L.Println("loadbalancer taken care of.")
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

func (l *LoadBalancer) Lbhandler(w http.ResponseWriter, r *http.Request) {
	// loadbalancer is essentially a reverse proxy
	// determine the scheme to be used ( http or https )
	// determine the backend url and port to forward to ( lb algo )
	// copy the request url path and query params
	// copy the headers except those that should not be copied as per RFC7230
	// copy request body

	scheme := os.Getenv("NBLB_SCHEME")
	targethost := l.Strategy.Next()
	targetURL := scheme + targethost + r.URL.RequestURI()

	request, err := http.NewRequestWithContext(r.Context(), r.Method, targetURL, r.Body)
	if err != nil {
		logger.L.Println("failed to create request")
		w.WriteHeader(http.StatusInternalServerError)
	}
	for k, v := range r.Header {
		if _, exists := consts.HopHeaders[k]; exists {
			continue
		} else {
			for _, j := range v {
				request.Header.Add(k, j)
			}
		}
	}

	resp, err := l.Client.Do(request)
	if err != nil {
		logger.L.Println("error during request execution : ", err)
	}

	for k, v := range resp.Header {
		for _, j := range v {
			w.Header().Add(k, j)
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)

}

func CheckEnvironment() error {
	missing := make([]string, 0, len(consts.RequiredEnvParameters))
	for k := range consts.RequiredEnvParameters {
		if os.Getenv(k) == "" {
			missing = append(missing, k)
		}
	}
	if len(missing) != 0 {
		return fmt.Errorf("required env parameters missing/not set: %v", missing)
	}
	return nil
}

func main() {

	err := logger.InitLogger(true, true)
	if err != nil {
		fmt.Println("error initialising logger: ", err)
		os.Exit(1)
	}

	_ = godotenv.Load()
	err = CheckEnvironment()
	if err != nil {
		logger.L.Fatalf("env check failed :%v", err)
	}

	var lb LoadBalancer
	err = lb.Initialise(os.Getenv("NBLB_PORT"), os.Getenv("NBLB_JSONPATH"))
	if err != nil {
		logger.L.Fatalf("error during initialisation : %v", err)
	}

	go func() {
		if err := lb.Start(); err != nil {
			logger.L.Fatalf("error during startup : %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop //listen for activity on channel; blocking

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := lb.Stop(ctx); err != nil {
		logger.L.Fatalf("error during shutdown : %v", err)
	}
}
