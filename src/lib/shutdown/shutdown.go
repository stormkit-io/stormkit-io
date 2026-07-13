package shutdown

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

var fns []func() error
var mux sync.Mutex

func init() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Cleanup on exit
	go func() {
		<-c
		Shutdown()
	}()
}

func Subscribe(fn func() error) {
	mux.Lock()
	defer mux.Unlock()
	fns = append(fns, fn)
}

func Shutdown() {
	fmt.Println("running clean up operations")

	for _, fn := range fns {
		if err := fn(); err != nil {
			fmt.Printf("error while shutting down: %s", err.Error())
			fmt.Println()
		}
	}

	os.Exit(0)
}
