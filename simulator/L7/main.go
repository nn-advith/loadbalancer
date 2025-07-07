package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func createRootHandler(address string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf("hit root %s", address)))
	}
}

func createHealthCheckHandler(address string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf("health OK: from %s", address)))
	}
}

type SimServer struct {
	Identity string
	Server   *http.Server
}

func (s *SimServer) Initialise(address string) error {

	handler := http.NewServeMux()
	handler.HandleFunc("/", createRootHandler(address))
	handler.HandleFunc("/health", createHealthCheckHandler(address))

	s.Identity = address
	s.Server = &http.Server{
		Addr:    address,
		Handler: handler,
	}
	return nil
}

func (s *SimServer) Start() error {
	fmt.Println("Simulator started and listening on : ", s.Server.Addr)
	if err := s.Server.ListenAndServe(); err != http.ErrServerClosed {
		fmt.Println("error during startup")
		return err
	}
	return nil
}

func (s *SimServer) Stop(context context.Context) error {
	fmt.Println("Beginning shutdown of server")
	if err := s.Server.Shutdown(context); err != nil {
		fmt.Println("error during shutdown")
		return err
	}
	fmt.Println("Sim server with ID: ", s.Identity, " shutdown")
	return nil
}

// group

type Simulators struct {
	Sims []*SimServer
}

func CreateSimulators(addresses []string) (*Simulators, error) {
	var SimSlice Simulators

	for i := range addresses {
		var ns SimServer
		na := ":" + addresses[i]
		if err := ns.Initialise(na); err != nil {
			return nil, fmt.Errorf("initialisation error for address %s: %v", na, err)
		}
		SimSlice.Sims = append(SimSlice.Sims, &ns)
	}

	return &SimSlice, nil
}

func (s *Simulators) StartAll() error {

	for i := range len(s.Sims) {
		i := i
		go func(i int) {
			if err := s.Sims[i].Start(); err != nil {
				log.Println(err)
			}
		}(i)
	}
	return nil
}

func (s *Simulators) StopAll(ctx context.Context) error {

	var wg sync.WaitGroup
	ech := make(chan error, len(s.Sims))

	for i := range s.Sims {
		wg.Add(1)
		i := i
		go func() {
			defer wg.Done()
			if err := s.Sims[i].Stop(ctx); err != nil {
				ech <- err
			}
		}()
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		close(ech)
		for err := range ech {
			return err
		}
		return nil
	}

}

func main() {
	// var address string
	// var ns SimServer

	// if err := ns.Initialise(address); err != nil {
	// 	log.Fatal(err)
	// }

	// go func() {
	// 	if err := ns.Start(); err != nil {
	// 		log.Fatal(err)
	// 	}
	// }()

	var addresses []string
	if len(os.Args) > 1 {
		// address = ":" + os.Args[1]
		addresses = os.Args[1:len(os.Args)]
	} else {
		log.Fatal("addresses not provided; exiting")
		return
	}

	simslice, err := CreateSimulators(addresses)
	if err != nil {
		log.Fatal(err)
	}

	if err := simslice.StartAll(); err != nil {
		log.Fatal(err)
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := simslice.StopAll(ctx); err != nil {
		log.Fatal("shutdown error", err)
	} else {
		fmt.Println("clean shutdown")
	}

}
