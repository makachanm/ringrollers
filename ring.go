package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// Token represents the message passed around the ring.
type Token struct {
	Issuer   string   `json:"issuer"`
	IssuedAt int64    `json:"issued_at"`
	Signers  []string `json:"signers"` // URLs of nodes. First entry is the issuer's URL.
}

// Ring represents a node in the ring.
type Ring struct {
	ID            string
	ListenAddr    string
	PublicAddr    string
	KnownNeighbor string
	status        *RingStatus // Share status with the web API
}

// NewRing creates a new Ring instance.
func NewRing(id, listenAddr, publicAddr, neighborAddr string, status *RingStatus) *Ring {
	return &Ring{
		ID:            id,
		ListenAddr:    listenAddr,
		PublicAddr:    publicAddr,
		KnownNeighbor: neighborAddr,
		status:        status,
	}
}

// RegisterHandler sets up the HTTP handler for internode communication.
func (r *Ring) RegisterHandler() {
	http.HandleFunc("/token", r.handleToken)
}

func (r *Ring) IssueAndPassToken() {
	log.Printf("Node %s is issuing a new token.", r.ID)
	token := &Token{
		Issuer:   r.ID,
		IssuedAt: time.Now().Unix(),
		Signers:  []string{r.PublicAddr}, // First entry is issuer's public URL
	}
	r.forwardToken(token)
}

func (r *Ring) handleToken(w http.ResponseWriter, req *http.Request) {
	var token Token
	if err := json.NewDecoder(req.Body).Decode(&token); err != nil {
		http.Error(w, "Invalid token format", http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	log.Printf("Node %s received token.", r.ID)

	if token.Issuer == r.ID {
		log.Printf("Token has completed its journey and returned to issuer %s.", r.ID)
		r.verifyRing(&token)
		w.WriteHeader(http.StatusOK)
		return
	}

	token.Signers = append(token.Signers, r.PublicAddr)
	r.forwardToken(&token)

	w.WriteHeader(http.StatusOK)
}

func (r *Ring) forwardToken(token *Token) {
	// This is the fallback function to return the token to the issuer.
	returnToIssuer := func() {
		if len(token.Signers) == 0 {
			log.Printf("Cannot return token to issuer: Signers list is empty.")
			return
		}
		issuerURL := fmt.Sprintf("%s/token", token.Signers[0])
		log.Printf("Returning token to issuer at %s.", issuerURL)

		jsonData, err := json.Marshal(token)
		if err != nil {
			log.Printf("Error marshalling token for issuer return: %v", err)
			return
		}

		resp, err := http.Post(issuerURL, "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			log.Printf("!!! Failed to return token to issuer %s: %v", issuerURL, err)
			return
		}
		defer resp.Body.Close()
		log.Printf("Token successfully returned to issuer.")
	}

	if r.KnownNeighbor == "" {
		log.Printf("Node %s has no neighbor.", r.ID)
		returnToIssuer()
		return
	}

	neighborURL := fmt.Sprintf("%s/token", r.KnownNeighbor)
	log.Printf("Node %s is attempting to forward token to neighbor %s.", r.ID, neighborURL)

	jsonData, err := json.Marshal(token)
	if err != nil {
		log.Printf("Error marshalling token: %v", err)
		return
	}

	// Attempt to send to neighbor
	resp, err := http.Post(neighborURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		// If neighbor is unreachable, send back to issuer
		log.Printf("!!! Neighbor %s is unreachable: %v", neighborURL, err)
		returnToIssuer()
		return
	}

	// If neighbor was reached successfully
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("Neighbor %s returned non-OK status: %s", neighborURL, resp.Status)
	} else {
		log.Printf("Token successfully forwarded to neighbor %s.", r.KnownNeighbor)
	}
}

func (r *Ring) verifyRing(token *Token) {
	log.Println("--- Ring Health Check Result ---")
	log.Printf("Token issued by %s.", token.Issuer)
	log.Printf("Path taken before returning: %v", token.Signers)
	log.Println("---------------------------------")

	// Update the shared status with the completed token
	if r.status != nil {
		r.status.Set(token)
	}
}
