package com.shinro.migration.omscamundaquest1.domain;

import com.fasterxml.jackson.annotation.JsonProperty;

public record PaymentInput(
    @JsonProperty("rrn") String rrn,
    @JsonProperty("amount") double amount) {}