package main

import (
	"log"
	"net/http"
)

func main() {
	c, err := NewInfraiClient()
	if err != nil {
		log.Fatal(err)
	}
	accounts := []Account{{TenantID: "acme", Active: true, Admin: true}, {TenantID: "closed", Active: false}}
	log.Println("snapshot service listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", serve(c, "saas-nightly", accounts)))
}
