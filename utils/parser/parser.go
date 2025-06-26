package parser

import (
	"encoding/json"
	"fmt"
	"os"
)

type Backend struct {
	Address string  `json:"address"`
	Port    string  `json:"port"`
	Weight  float64 `json:"weight"`
}

type Backends struct {
	Backends []Backend `json:"backends"`
}

func ParseBackendJSON() ([]Backend, error) {

	var parsedBackends Backends

	backendjsonpath := os.Getenv("NBLB_JSONPATH")
	if backendjsonpath == "" {
		return nil, fmt.Errorf("env NBLB_JSONPATH is not set")
	}

	if _, err := os.Stat(backendjsonpath); os.IsNotExist(err) {
		return nil, fmt.Errorf("backend json config not found: %v", err)
	}
	data, err := os.ReadFile(backendjsonpath)
	if err != nil {
		return nil, fmt.Errorf("unable to read json config : %v", err)
	}

	if err := json.Unmarshal(data, &parsedBackends); err != nil {
		return nil, fmt.Errorf("error during unmarshalling json config: %v", err)
	}

	if len(parsedBackends.Backends) == 0 {
		return nil, fmt.Errorf("no backends defined")
	}

	return parsedBackends.Backends, err
}

func ConstructBackendURIs(backends []Backend) []string {
	bl := make([]string, 0, len(backends))
	for i := range backends {
		bl = append(bl, fmt.Sprintf("%s:%s", backends[i].Address, backends[i].Port))
	}
	return bl
}

func GetWeights(backends []Backend) []float64 {
	bw := make([]float64, 0, len(backends))
	for i := range backends {
		bw = append(bw, backends[i].Weight)
	}
	return bw
}
