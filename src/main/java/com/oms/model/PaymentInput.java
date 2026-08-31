package com.oms.model;

import com.fasterxml.jackson.annotation.JsonProperty;

/**
 * Payment details captured via the {@code CapturePayment} message and validated
 * by the {@code validate-payment} worker. Mirrors {@code models.PaymentInput}.
 */
public record PaymentInput(
    @JsonProperty("rrn") String rrn,
    @JsonProperty("amount") double amount,
    @JsonProperty("currency") String currency,
    @JsonProperty("payment_method") String paymentMethod,
    @JsonProperty("paid_at") String paidAt) {}
