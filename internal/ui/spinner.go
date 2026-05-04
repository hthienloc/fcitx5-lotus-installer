package ui

import (
	"fmt"
	"time"
)

var spinnerChars = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

type Spinner struct {
	message string
	stop    chan bool
}

func NewSpinner(msg string) *Spinner {
	return &Spinner{
		message: msg,
		stop:    make(chan bool, 1),
	}
}

func (s *Spinner) Start() {
	go func() {
		i := 0
		for {
			select {
			case <-s.stop:
				return
			default:
				fmt.Printf("\r  %s %s", spinnerChars[i%len(spinnerChars)], s.message)
				time.Sleep(100 * time.Millisecond)
				i++
			}
		}
	}()
}

func (s *Spinner) Stop(success bool) {
	s.stop <- true
	fmt.Printf("\r")
	if success {
		fmt.Println("  ✓ " + s.message)
	} else {
		fmt.Println("  ✗ " + s.message)
	}
}
