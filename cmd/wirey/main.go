package main

import (
	"flag"
	"log"
	"net/http"
	_ "net/http/pprof"
)

func main() {
	enableProfiling := flag.Bool("enable-profiling", false, "Enable pprof profiling")
	profileAddr := flag.String("profile-addr", "localhost:6060", "Address for pprof profiling")

	flag.Parse()

	if *enableProfiling {
		go func() {
			log.Printf("Starting pprof server on %s\n", *profileAddr)
			if err := http.ListenAndServe(*profileAddr, nil); err != nil {
				log.Fatalf("Error starting pprof server: %v\n", err)
			}
		}()
	}

	// Execute main application logic
	Execute()
}
