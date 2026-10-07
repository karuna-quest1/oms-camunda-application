// Command starter starts a workflow instance of process "order-processing" —
// the Temporal parity of the source entry point POST /orders
// (OrderController#submitOrder, src/main/java/com/oms/api/OrderController.java:106),
// which creates the Zeebe instance with start variables orderId + order.
//
// Usage: starter <orderId> <orderJSON>
//
// The workflow id equals the orderId — the parity of the source's Zeebe
// message correlationKey=orderId: signals (Message_CapturePayment,
// Message_SupportCorrection, Message_CancelOrder) are sent to that workflow id.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"go.temporal.io/sdk/client"

	"shinro-migration/oms-camunda-application/shared"
	"shinro-migration/oms-camunda-application/workflows"
)

func main() {
	if len(os.Args) != 3 {
		log.Fatalf("usage: %s <orderId> <orderJSON>", os.Args[0])
	}
	orderID := os.Args[1]
	var order shared.OrderInput
	if err := json.Unmarshal([]byte(os.Args[2]), &order); err != nil {
		log.Fatalf("invalid order JSON: %v", err)
	}

	addr := os.Getenv("TEMPORAL_ADDRESS")
	if addr == "" {
		addr = client.DefaultHostPort
	}
	c, err := client.Dial(client.Options{HostPort: addr})
	if err != nil {
		log.Fatalln("unable to create Temporal client", err)
	}
	defer c.Close()

	// ExecuteWorkflow blocks until the order reaches a terminal state; it is
	// the version-agnostic starter API in this SDK build.
	run, err := c.ExecuteWorkflow(
		context.Background(),
		client.StartWorkflowOptions{ID: orderID, TaskQueue: "order-processing"},
		workflows.OrderProcessingWorkflow,
		shared.OrderProcessingInput{OrderId: orderID, Order: &order},
	)
	if err != nil {
		log.Fatalf("failed to start workflow: %v", err)
	}
	var result shared.OrderProcessingResult
	if err := run.Get(context.Background(), &result); err != nil {
		log.Fatalf("workflow failed: %v", err)
	}
	fmt.Printf("order %s finished: %s\n", orderID, result.Outcome)
}
