// Package shared holds the payload types exchanged between workflows,
// activities, signals, and starters. The JSON names mirror the Zeebe
// process-variable wire format of the source application
// (com.oms.model.* Jackson @JsonProperty names) so payloads produced by
// the original REST gateway remain compatible.
package shared

import "time"

// OrderItem is a single order line item (com.oms.model.OrderItem).
// SKUId / BrandCode are populated by the PIM enrichment step; they are
// absent on inbound requests.
type OrderItem struct {
	ItemId     string  `json:"item_id"`
	Quantity   int     `json:"quantity"`
	UnitPrice  float64 `json:"unit_price"`
	SkuId      string  `json:"sku_id,omitempty"`
	BrandCode  string  `json:"brand_code,omitempty"`
}

// OrderInput is the canonical order payload carried as the "order"
// process variable (com.oms.model.OrderInput).
type OrderInput struct {
	OrderId     string      `json:"order_id"`
	CustomerId  string      `json:"customer_id"`
	Items       []OrderItem `json:"items"`
	TotalAmount float64     `json:"total_amount"`
	Currency    string      `json:"currency"`
}

// TotalAmount returns the order total, or 0 when the order is nil
// (parity with the null-tolerant read in the validate-payment worker).
func (o *OrderInput) TotalAmountOrZero() float64 {
	if o == nil {
		return 0
	}
	return o.TotalAmount
}

// PaymentInput is the payment captured via the CapturePayment message
// (com.oms.model.PaymentInput).
type PaymentInput struct {
	Rrn           string  `json:"rrn"`
	Amount        float64 `json:"amount"`
	Currency      string  `json:"currency"`
	PaymentMethod string  `json:"payment_method"`
	PaidAt        string  `json:"paid_at"`
}

// FulfillmentMessage is published to the downstream fulfillment topic by
// the publish-fulfillment worker (com.oms.model.FulfillmentMessage).
// Consumers must deduplicate on EventId — delivery is at-least-once.
type FulfillmentMessage struct {
	EventId       string        `json:"event_id"`
	SchemaVersion int           `json:"schema_version"`
	CustomerId    string        `json:"customer_id"`
	OrderId       string        `json:"order_id"`
	Payment       *PaymentInput `json:"payment_details"`
	Items         []OrderItem   `json:"items"`
}

// OrderStatus is the customer-facing lifecycle state exposed by the
// dashboard read model and the status endpoint (com.oms.model.OrderStatus).
type OrderStatus string

const (
	StatusOrderReceived       OrderStatus = "ORDER_RECEIVED"
	StatusAwaitingCorrection  OrderStatus = "AWAITING_CORRECTION"
	StatusPaymentPending      OrderStatus = "PAYMENT_PENDING"
	StatusPaymentCaptured     OrderStatus = "PAYMENT_CAPTURED"
	StatusFulfilled           OrderStatus = "FULFILLED"
	StatusCancelled           OrderStatus = "CANCELLED"
	StatusExpired             OrderStatus = "EXPIRED"
	StatusFailedPayment       OrderStatus = "FAILED_PAYMENT"
)

// validOrderStatuses mirrors the OrderStatus enum; anything else is an
// unsupported dashboard status (Java OrderStatus.fromString throws).
var validOrderStatuses = map[OrderStatus]bool{
	StatusOrderReceived:    true,
	StatusAwaitingCorrection: true,
	StatusPaymentPending:   true,
	StatusPaymentCaptured:  true,
	StatusFulfilled:        true,
	StatusCancelled:        true,
	StatusExpired:          true,
	StatusFailedPayment:    true,
}

// Valid reports whether s is one of the known OrderStatus values.
func (s OrderStatus) Valid() bool { return validOrderStatuses[s] }

// Workflow / activity names (BPMN job types and error codes are the
// single source of coupling, per ProcessConstants.java).
const (
	// WorkflowType is the Temporal workflow name for process "order-processing".
	WorkflowType = "order_processing"

	JobValidateOrder      = "validate-order"
	JobValidatePayment    = "validate-payment"
	JobEnrichOrder        = "enrich-order"
	JobUpdateDashboard    = "update-dashboard"
	JobPublishFulfillment = "publish-fulfillment"

	ErrInvalidOrder  = "INVALID_ORDER"
	ErrInvalidPayment = "INVALID_PAYMENT"

	// OrderTTL is the 30-day lifetime enforced by the interrupting TTL
	// event subprocess (Start_TTL, timeDuration=PT720H).
	OrderTTL = 720 * time.Hour

	// Signal names — one Temporal signal per BPMN message, correlated on
	// the workflow id (parity of correlationKey=orderId).
	SignalCapturePayment = "Message_CapturePayment"
	SignalSupportCorrection = "Message_SupportCorrection"
	SignalCancelOrder    = "Message_CancelOrder"
)

// OrderProcessingInput is the start payload of OrderProcessingWorkflow
// (the variables of StartEvent_1: "orderId" and "order").
type OrderProcessingInput struct {
	OrderId string      `json:"orderId"`
	Order   *OrderInput `json:"order"`
}

// OrderProcessingResult is the terminal result of the workflow. Outcome
// is one of "FULFILLED", "CANCELLED", "EXPIRED" (the three BPMN end events).
type OrderProcessingResult struct {
	OrderId     string        `json:"orderId"`
	Outcome     string        `json:"outcome"`
	Order       *OrderInput   `json:"order,omitempty"`
	Payment     *PaymentInput `json:"payment,omitempty"`
}

// CapturePaymentSignal carries the "payment" variable published by
// Message_CapturePayment.
type CapturePaymentSignal struct {
	OrderId string        `json:"orderId"`
	Payment *PaymentInput `json:"payment"`
}

// SupportCorrectionSignal carries the "correctionItems" variable
// published by Message_SupportCorrection.
type SupportCorrectionSignal struct {
	OrderId     string        `json:"orderId"`
	Items       []OrderItem   `json:"correctionItems"`
}

// CancelOrderSignal is the payload of Message_CancelOrder
// (CancelWebhookRequest: orderId + reason).
type CancelOrderSignal struct {
	OrderId string `json:"orderId"`
	Reason  string `json:"reason"`
}

// --- activity inputs / outputs -------------------------------------------

// ValidateOrderInput is the input of activity ValidateOrder
// (job type validate-order). CorrectionItems is the pending
// SupportCorrection delivery (may be nil on first validation).
type ValidateOrderInput struct {
	Order           *OrderInput `json:"order"`
	CorrectionItems []OrderItem `json:"correctionItems,omitempty"`
}

// ValidateOrderOutput is the output of activity ValidateOrder.
type ValidateOrderOutput struct {
	Order *OrderInput `json:"order"`
}

// EnrichOrderInput is the input of activity EnrichOrder (job type enrich-order).
type EnrichOrderInput struct {
	Order *OrderInput `json:"order"`
}

// EnrichOrderOutput is the output of activity EnrichOrder.
type EnrichOrderOutput struct {
	Order *OrderInput `json:"order"`
}

// PublishFulfillmentInput is the input of activity PublishFulfillment
// (job type publish-fulfillment).
type PublishFulfillmentInput struct {
	Order   *OrderInput   `json:"order"`
	Payment *PaymentInput `json:"payment"`
}

// PublishFulfillmentOutput is the output of activity PublishFulfillment.
type PublishFulfillmentOutput struct {
	EventId string `json:"eventId"`
}

// UpdateDashboardInput is the input of activity UpdateDashboard
// (job type update-dashboard). Status is the per-task "status" job
// header in BPMN — a single activity serves every dashboard step.
type UpdateDashboardInput struct {
	OrderId string      `json:"orderId"`
	Status  OrderStatus `json:"status"`
}

// UpdateDashboardOutput is the output of activity UpdateDashboard.
type UpdateDashboardOutput struct {
	Applied bool `json:"applied"`
}

// ValidatePaymentInput is the input of activity ValidatePayment
// (job type validate-payment).
type ValidatePaymentInput struct {
	Order   *OrderInput   `json:"order"`
	Payment *PaymentInput `json:"payment"`
}

// ValidatePaymentOutput is the output of activity ValidatePayment.
type ValidatePaymentOutput struct {
	Payment *PaymentInput `json:"payment"`
}
