// Command starter starts workflow instances — one function per Camunda entry
// point discovered by the analyzer. main wires up the first one as an example;
// call the others from your own entry points.
package main

import (
	"log"
	"os"

	"go.temporal.io/sdk/client"

	_ "shinro-migration/oms-camunda-quest1/shared"
	_ "shinro-migration/oms-camunda-quest1/workflows"
)

func main() {
	addr := os.Getenv("TEMPORAL_ADDRESS")
	if addr == "" {
		addr = client.DefaultHostPort
	}
	c, err := client.Dial(client.Options{HostPort: addr})
	if err != nil {
		log.Fatalln("unable to create Temporal client", err)
	}
	defer c.Close()

	log.Println("no entry points were found; call the starter functions from your own code")
}
