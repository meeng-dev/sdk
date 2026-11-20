// Adds a sending domain, prints the DNS records to publish and asks for them
// to be checked.
//
//	MEENG_API_KEY=mg_xxx go run ./examples/add_domain
package main

import (
	"context"
	"fmt"
	"log"
	"strconv"

	meeng "github.com/meeng-dev/sdk/go"
)

func main() {
	client := meeng.New("")
	ctx := context.Background()

	domain, err := client.Domains.Create(ctx, meeng.DomainsCreateParams{Name: "acme.com.br", Region: "us-east-1"})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("publique estes registros no DNS de", domain.Name)
	for _, record := range domain.DNSRecords {
		fmt.Printf("  %-5s %-40s %s\n", record.Type, record.Name, record.Value)
	}

	id := strconv.FormatInt(domain.ID, 10)
	if _, err := client.Domains.Verify(ctx, id); err != nil {
		log.Fatal(err)
	}
	checked, err := client.Domains.Get(ctx, id)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("situação:", checked.Status)
}
