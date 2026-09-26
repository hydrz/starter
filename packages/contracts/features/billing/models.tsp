import "@typespec/openapi";

namespace Starter.Billing;

@friendlyName("SubscriptionStatus")
enum SubscriptionStatus {
  active,
  trialing,
  past_due,
  canceled,
  incomplete,
  incomplete_expired,
  unpaid,
  paused,
}

@friendlyName("SubscriptionSummary")
model SubscriptionSummary {
  @format("uuid")
  id: string;

  priceKey: string;
  status: SubscriptionStatus;
  currentPeriodEnd?: utcDateTime;
  cancelAtPeriodEnd: boolean;
}

@friendlyName("EntitlementSummary")
model EntitlementSummary {
  featureKey: string;
  enabled: boolean;
  expiresAt?: utcDateTime;
}

@friendlyName("BillingSummary")
model BillingSummary {
  @format("uuid")
  organizationId: string;

  hasActiveSubscription: boolean;
  subscription?: SubscriptionSummary;
  entitlements: EntitlementSummary[];
}

@friendlyName("CreateCheckoutSessionInput")
model CreateCheckoutSessionInput {
  // Server-controlled catalog key only. The server maps this to a real
  // Stripe Price ID; clients can never supply a Stripe price/customer/
  // subscription ID, an amount, or a success/cancel URL.
  @minLength(1)
  @maxLength(100)
  priceKey: string;
}

@friendlyName("CheckoutSessionResponse")
model CheckoutSessionResponse {
  @format("uuid")
  id: string;

  url: string;
}

@friendlyName("PortalSessionResponse")
model PortalSessionResponse {
  url: string;
}
