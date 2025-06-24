package logger

import (
	"io"
	"log"
	"os"
	"sync"
)

var (
	L      *log.Logger
	once   sync.Once
	logdir string
)

func createLogDirectory() error {
	//create a log directory on windows or linux keep it handy
	return nil
}

func InitLogger(filelog bool, stdlog bool, logfilepath *string) error {
	var gerr error
	once.Do(func() {
		var writers []io.Writer

		if filelog {
			// add a file pointer to the writers
			file, err := os.OpenFile(*logfilepath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
			if err != nil {
				gerr = err
			} else {
				writers = append(writers, file)
			}
		}

		if stdlog {
			writers = append(writers, os.Stdout)
		}

		multiop := io.MultiWriter(writers...)
		L = log.New(multiop, "[NB-LB]:", log.LstdFlags)
	})
	return gerr
}
