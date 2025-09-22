package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	id := flag.String("id", "", "ID of this node (required)")
	addr := flag.String("addr", ":8080", "Address to listen on (e.g., :8080)")
	publicAddr := flag.String("public.addr", "", "Publicly reachable URL (optional, defaults to http://localhost + addr)")
	neighbor := flag.String("neighbor", "", "Full address of the next node in the ring (e.g., http://localhost:8081)")
	isInitiator := flag.Bool("initiator", false, "Set to true if this node should start the token passing")
	flag.Parse()

	if *id == "" {
		log.Println("Missing required flag: -id must be provided.")
		flag.Usage()
		os.Exit(1)
	}

	if *publicAddr == "" {
		*publicAddr = fmt.Sprintf("http://localhost%s", *addr)
	}

	if *neighbor == "" {
		log.Println("No neighbor specified. This node is the end of the chain.")
	}

	log.Printf("Starting node with config: id=%s, addr=%s, public.addr=%s, neighbor=%s, initiator=%t", *id, *addr, *publicAddr, *neighbor, *isInitiator)

	// 1. Create shared status holder
	status := &RingStatus{}

	// 2. Create the ring logic instance, passing the status holder
	ring := NewRing(*id, *addr, *publicAddr, *neighbor, status)
	ring.RegisterHandler() // Register /token handler

	// 3. Create the web API instance, also passing the status holder
	webAPI := NewWebAPI(status)
	webAPI.RegisterHandler() // Register /status handler

	// 4. Start the token passing goroutine if this is the initiator
	if *isInitiator {
		go func() {
			time.Sleep(3 * time.Second) // Wait a moment for other servers to start
			ticker := time.NewTicker(15 * time.Second)
			defer ticker.Stop()

			// Issue the first token immediately
			ring.IssueAndPassToken()

			for range ticker.C {
				ring.IssueAndPassToken()
			}
		}()
	}

	// 5. Start the single HTTP server
	log.Printf("Starting server on %s. API endpoint: /status, P2P endpoint: /token", *addr)
	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
