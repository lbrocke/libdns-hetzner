package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/libdns/hetzner"
)

func main() {
	token := os.Getenv("LIBDNS_HETZNER_TOKEN")
	zone := os.Getenv("LIBDNS_HETZNER_ZONE")

	if token == "" || zone == "" {
		fmt.Printf("LIBDNS_HETZNER_TOKEN and/or LIBDNS_HETZNER_ZONE not set\n")
		return
	}

	p := hetzner.New(token)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()
	records, err := p.GetRecords(ctx, zone)

	if err != nil {
		fmt.Printf("Error: %s", err.Error())
		return
	}

	for _, record := range records {
		fmt.Println(record)
	}
}
