package activities

import (
	"context"
	"testing"

	"go.temporal.io/sdk/temporal"

	"shinro-migration/oms-camunda-application/shared"
)

var orderWithItems = &shared.OrderInput{
	OrderId:     "ord-1",
	CustomerId:  "cust-1",
	Items:       []shared.OrderItem{{ItemId: "i1", Quantity: 1, UnitPrice: 5.0}},
	TotalAmount: 5.0,
	Currency:    "USD",
}

func TestValidateOrderValidNoCorrection(t *testing.T) {
	out, err := ValidateOrder(context.Background(),
		shared.ValidateOrderInput{Order: orderWithItems, CorrectionItems: nil})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Order == nil || out.Order.OrderId != "ord-1" {
		t.Fatalf("output order = %+v, want the input order echoed", out.Order)
	}
}

func TestValidateOrderAppliesCorrection(t *testing.T) {
	bad := &shared.OrderInput{
		OrderId:     "ord-1",
		CustomerId:  "cust-1",
		Items:       nil,
		TotalAmount: 5.0,
		Currency:    "USD",
	}
	corrected := []shared.OrderItem{{ItemId: "i1", Quantity: 1, UnitPrice: 5.0}}

	out, err := ValidateOrder(context.Background(),
		shared.ValidateOrderInput{Order: bad, CorrectionItems: corrected})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The correction replaces the items, keeping the other fields
	// (parity of applyCorrection).
	if len(out.Order.Items) != 1 || out.Order.Items[0].ItemId != "i1" {
		t.Fatalf("items = %+v, want the corrected items", out.Order.Items)
	}
	if out.Order.TotalAmount != 5.0 || out.Order.CustomerId != "cust-1" {
		t.Fatalf("order = %+v, want total/customer preserved", out.Order)
	}
}

func TestValidateOrderRejectsInvalid(t *testing.T) {
	cases := []struct {
		name    string
		order   *shared.OrderInput
		message string
	}{
		{"nil order", nil, "order_id is required"},
		{"blank order id", &shared.OrderInput{OrderId: " ", Items: orderWithItems.Items, TotalAmount: 1, CustomerId: "c"}, "order_id is required"},
		{"no items", &shared.OrderInput{OrderId: "o", Items: nil, TotalAmount: 1, CustomerId: "c"}, "order must contain at least one item"},
		{"zero total", &shared.OrderInput{OrderId: "o", Items: orderWithItems.Items, TotalAmount: 0, CustomerId: "c"}, "invalid total_amount: 0"},
		{"blank customer", &shared.OrderInput{OrderId: "o", Items: orderWithItems.Items, TotalAmount: 1, CustomerId: ""}, "customer_id is required"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ValidateOrder(context.Background(),
				shared.ValidateOrderInput{Order: tc.order})
			if err == nil {
				t.Fatal("expected an error")
			}
			ae, ok := err.(*temporal.ApplicationError)
			if !ok {
				t.Fatalf("error type = %T, want *temporal.ApplicationError", err)
			}
			if ae.Type() != shared.ErrInvalidOrder {
				t.Fatalf("error code = %q, want %s", ae.Type(), shared.ErrInvalidOrder)
			}
			if !ae.NonRetryable() {
				t.Fatal("INVALID_ORDER must be non-retryable (drives the correction loop, not a retry)")
			}
			if ae.Message() != tc.message {
				t.Fatalf("message = %q, want %q", ae.Message(), tc.message)
			}
		})
	}
}

func TestValidatePaymentValid(t *testing.T) {
	payment := &shared.PaymentInput{Rrn: "123456789012", Amount: 5.0, Currency: "USD"}
	out, err := ValidatePayment(context.Background(),
		shared.ValidatePaymentInput{Order: orderWithItems, Payment: payment})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Payment == nil || out.Payment.Rrn != payment.Rrn {
		t.Fatalf("payment = %+v, want the validated payment echoed", out.Payment)
	}
}

func TestValidatePaymentRejects(t *testing.T) {
	mk := func(p *shared.PaymentInput) *shared.PaymentInput { return p }
	cases := []struct {
		name    string
		payment *shared.PaymentInput
		message string
	}{
		{"nil payment", nil, "payment RRN is empty"},
		{"empty rrn", mk(&shared.PaymentInput{Amount: 5.0}), "payment RRN is empty"},
		{"short rrn", mk(&shared.PaymentInput{Rrn: "123", Amount: 5.0}), "invalid RRN length 3; must be 12 digits"},
		{"non-digit rrn", mk(&shared.PaymentInput{Rrn: "12345678901a", Amount: 5.0}), "RRN must contain only digits"},
		{"negative amount", mk(&shared.PaymentInput{Rrn: "123456789012", Amount: -1}), "invalid payment amount: -1"},
		{"amount mismatch", mk(&shared.PaymentInput{Rrn: "123456789012", Amount: 6.0}), "payment amount 6.00 does not match order total 5.00"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ValidatePayment(context.Background(),
				shared.ValidatePaymentInput{Order: orderWithItems, Payment: tc.payment})
			if err == nil {
				t.Fatal("expected an error")
			}
			ae, ok := err.(*temporal.ApplicationError)
			if !ok {
				t.Fatalf("error type = %T, want *temporal.ApplicationError", err)
			}
			if ae.Type() != shared.ErrInvalidPayment {
				t.Fatalf("error code = %q, want %s", ae.Type(), shared.ErrInvalidPayment)
			}
			if !ae.NonRetryable() {
				t.Fatal("INVALID_PAYMENT must be non-retryable (drives the boundary loop)")
			}
			if ae.Message() != tc.message {
				t.Fatalf("message = %q, want %q", ae.Message(), tc.message)
			}
		})
	}
}

func TestEnrichOrderFillsMissingMetadata(t *testing.T) {
	in := &shared.OrderInput{
		OrderId:     "ord-1",
		CustomerId:  "cust-1",
		Items: []shared.OrderItem{
			{ItemId: "i1", Quantity: 1, UnitPrice: 5.0},
			{ItemId: "i2", Quantity: 2, UnitPrice: 3.0, SkuId: "SKU-GIVEN", BrandCode: "BRAND-GIVEN"},
		},
		TotalAmount: 11.0,
		Currency:    "USD",
	}
	out, err := EnrichOrder(context.Background(), shared.EnrichOrderInput{Order: in})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	o := out.Order
	if o.OrderId != "ord-1" || o.TotalAmount != 11.0 || o.Currency != "USD" {
		t.Fatalf("order fields changed: %+v", o)
	}
	if o.Items[0].SkuId != "SKU-ord-1-1" || o.Items[0].BrandCode != "BRAND-i1" {
		t.Fatalf("item 1 = %+v, want SKU-ord-1-1 / BRAND-i1", o.Items[0])
	}
	if o.Items[1].SkuId != "SKU-GIVEN" || o.Items[1].BrandCode != "BRAND-GIVEN" {
		t.Fatalf("item 2 = %+v, want the given sku/brand preserved", o.Items[1])
	}
}

func TestEnrichOrderKeepsExistingBlankSkuRegeneration(t *testing.T) {
	// A blank sku is regenerated, exactly like the Java isBlank() check.
	in := &shared.OrderInput{
		OrderId:  "ord-9",
		Items:    []shared.OrderItem{{ItemId: "x", Quantity: 1, UnitPrice: 1, SkuId: "  ", BrandCode: ""}},
		Currency: "USD",
	}
	out, err := EnrichOrder(context.Background(), shared.EnrichOrderInput{Order: in})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Order.Items[0].SkuId != "SKU-ord-9-1" || out.Order.Items[0].BrandCode != "BRAND-x" {
		t.Fatalf("item = %+v, want regenerated sku/brand", out.Order.Items[0])
	}
}
