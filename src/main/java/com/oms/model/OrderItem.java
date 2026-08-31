package com.oms.model;

import com.fasterxml.jackson.annotation.JsonProperty;

/**
 * A single order line item. {@code skuId} / {@code brandCode} are populated by
 * the PIM enrichment worker; they are absent on inbound requests.
 *
 * <p>Serialized into Zeebe process variables and into the fulfillment message,
 * so the JSON property names match the Temporal wire format.
 */
public record OrderItem(
    @JsonProperty("item_id") String itemId,
    @JsonProperty("quantity") int quantity,
    @JsonProperty("unit_price") double unitPrice,
    @JsonProperty("sku_id") String skuId,
    @JsonProperty("brand_code") String brandCode) {}
