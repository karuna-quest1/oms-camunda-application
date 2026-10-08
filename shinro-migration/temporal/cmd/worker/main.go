// Command worker runs the Temporal workers for every migrated process.
//
// Environment:
//
//	TEMPORAL_ADDRESS   host:port of the Temporal frontend (default localhost:7233)
//
// Wiring note: activities.UpdateDashboard and activities.PublishFulfillment
// delegate to DashboardWriter / FulfillmentPublisher interfaces. The
// defaults fail loudly with NOT_IMPLEMENTED (dashboard) or log a mocked
// publish (fulfillment, parity of the source worker's log-only publish).
// For production traffic, install real implementations before Run() —
// e.g. activities.SetDashboardWriter(activities.NewSQLDashboardWriter(db))
// for the JDBC dashboard projection, and activities.SetFulfillmentPublisher
// for a real broker client (consumers dedupe on the event id).
package main

import (
	"log"
	"os"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"shinro-migration/oms-camunda-application/activities"
	"shinro-migration/oms-camunda-application/workflows"
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

	interrupt := worker.InterruptCh()

	errCh := make(chan error, 1)

	go func() {
		w := worker.New(c, "order-processing", worker.Options{})
		w.RegisterWorkflow(workflows.OrderProcessingWorkflow)
		w.RegisterActivity(activities.ValidateOrder)
		w.RegisterActivity(activities.EnrichOrder)
		w.RegisterActivity(activities.PublishFulfillment)
		w.RegisterActivity(activities.UpdateDashboard)
		w.RegisterActivity(activities.ValidatePayment)
		errCh <- w.Run(interrupt)
	}()

	if err := <-errCh; err != nil {
		log.Fatalln("worker failed", err)
	}
}
