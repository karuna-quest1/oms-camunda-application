// Package shared holds the payload types exchanged between workflows,
// activities, signals, and starters. Field types are inferred from the
// analysis findings (and the cluster assessment when available); `any`
// means the type could not be determined statically.
package shared

import "time"

// OrderProcessingInput is the start payload of OrderProcessingWorkflow.
type OrderProcessingInput struct {
	OrderID    string      `json:"order_id"`
	CustomerID string      `json:"customer_id"`
	Items      []OrderItem `json:"items"`
	TotalAmount float64    `json:"total_amount"`
	Currency   string      `json:"currency"`
}

// OrderProcessingResult is the completed result of OrderProcessingWorkflow.
type OrderProcessingResult struct {
	Status  string `json:"status"`
	OrderID string `json:"order_id"`
}

// OrderItem represents a single order line item.
type OrderItem struct {
	ItemID    string  `json:"item_id"`
	Quantity  int     `json:"quantity"`
	UnitPrice float64 `json:"unit_price"`
	SkuID     string  `json:"sku_id,omitempty"`
	BrandCode string  `json:"brand_code,omitempty"`
}

// OrderInput is the enriched order data structure passed between activities.
type OrderInput struct {
	OrderID     string      `json:"order_id"`
	CustomerID  string      `json:"customer_id"`
	Items       []OrderItem `json:"items"`
	TotalAmount float64     `json:"total_amount"`
	Currency    string      `json:"currency"`
}

// PaymentInput represents payment details captured via CapturePayment signal.
type PaymentInput struct {
	RRN           string  `json:"rrn"`
	Amount        float64 `json:"amount"`
	Currency      string  `json:"currency"`
	PaymentMethod string  `json:"payment_method"`
	PaidAt        string  `json:"paid_at"`
}

// ValidateOrderInput is the input of activity ValidateOrder (job type validate-order).
type ValidateOrderInput struct {
	Order           OrderInput  `json:"order"`
	CorrectionItems []OrderItem `json:"correction_items,omitempty"`
}

// ValidateOrderOutput is the output of activity ValidateOrder.
type ValidateOrderOutput struct {
	Order OrderInput `json:"order"`
}

// EnrichOrderInput is the input of activity EnrichOrder (job type enrich-order).
type EnrichOrderInput struct {
	Order OrderInput `json:"order"`
}

// EnrichOrderOutput is the output of activity EnrichOrder.
type EnrichOrderOutput struct {
	Order OrderInput `json:"order"`
}

// PublishFulfillmentInput is the input of activity PublishFulfillment (job type publish-fulfillment).
type PublishFulfillmentInput struct {
	Order   OrderInput   `json:"order"`
	Payment PaymentInput `json:"payment"`
}

// PublishFulfillmentOutput is the output of activity PublishFulfillment.
type PublishFulfillmentOutput struct {
}

// UpdateDashboardInput is the input of activity UpdateDashboard (job type update-dashboard).
type UpdateDashboardInput struct {
	OrderID string `json:"order_id"`
	Status  string `json:"status"`
}

// UpdateDashboardOutput is the output of activity UpdateDashboard.
type UpdateDashboardOutput struct {
}

// ValidatePaymentInput is the input of activity ValidatePayment (job type validate-payment).
type ValidatePaymentInput struct {
	Payment        PaymentInput `json:"payment"`
	ExpectedAmount float64      `json:"expected_amount"`
}

// ValidatePaymentOutput is the output of activity ValidatePayment.
type ValidatePaymentOutput struct {
	Payment PaymentInput `json:"payment"`
}

// CapturePaymentSignal is the payload of the signal for capturing payment.
type CapturePaymentSignal struct {
	Payment PaymentInput `json:"payment"`
}

// CancelOrderSignal is the payload of the signal for cancelling an order.
type CancelOrderSignal struct {
}

// SupportCorrectionSignal is the payload of the signal for applying support corrections.
type SupportCorrectionSignal struct {
	CorrectionItems []OrderItem `json:"correction_items"`
}

// DashboardStatus represents a dashboard projection row.
type DashboardStatus struct {
	OrderID   string    `json:"order_id"`
	Status    string    `json:"status"`
	UpdatedAt time.Time `json:"updated_at"`
}

