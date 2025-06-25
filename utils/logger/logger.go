package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

var (
	L      *log.Logger
	once   sync.Once
	logdir string
)

func createLogDirectory() error {
	//create a log directory on windows or linux keep it handy
	var LOGDIR string
	if runtime.GOOS == "windows" {
		//create log directory in roaming directory of current user
		// can remove this because why would you run a loadbalancer on a windows os
		LOGDIR = filepath.Join(os.Getenv("APPDATA"), "nbloadbalancer", "logs")
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("unable to get the home directory of user: %v", err)
		}
		LOGDIR = filepath.Join(home, "nbloadbalancer", "logs")
	}

	// check if logdir exists; if not > create

	if f, err := os.Stat(LOGDIR); os.IsNotExist(err) {
		//directory not present
		err := os.MkdirAll(LOGDIR, 0755)
		if err != nil {
			return fmt.Errorf("error during directory  creation: %v", err)
		}
	} else if err != nil {
		return fmt.Errorf("error during log directory check: %v", err)
	} else if !f.IsDir() {
		return fmt.Errorf("%v  is not a directory", LOGDIR)
	}
	logdir = LOGDIR
	return nil
}

func InitLogger(filelog bool, stdlog bool) error {
	var gerr error
	once.Do(func() {
		var writers []io.Writer
		err := createLogDirectory()
		if err != nil {
			gerr = err
			return
		}
		logfilepath := fmt.Sprintf("%v/log-%v.log", logdir, strings.Split(time.Now().Format(time.RFC3339), "T")[0])
		if filelog {
			// add a file pointer to the writers
			file, err := os.OpenFile(logfilepath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
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
