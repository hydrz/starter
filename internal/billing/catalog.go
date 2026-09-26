package billing

import (
	"encoding/json"
	"fmt"
)

// CatalogEntry maps one server-controlled price key to the real Stripe
// object(s) it purchases and the feature it grants. Clients only ever send
// the price key (see CreateCheckoutSessionInput in the billing TypeSpec
// contract); they can never supply a Stripe price ID, an amount, or a
// customer/subscription ID directly.
type CatalogEntry struct {
	// StripePriceID is the real Stripe Price object ID this key resolves to.
	StripePriceID string `json:"stripePriceId"`
	// FeatureKey is the entitlement this price grants when paid.
	FeatureKey string `json:"featureKey"`
	// Mode is "subscription" for recurring prices or "payment" for one-time
	// prices, matching Stripe Checkout Session's mode parameter.
	Mode string `json:"mode"`
}

const (
	ModeSubscription = "subscription"
	ModePayment      = "payment"
)

// Catalog maps price keys to their CatalogEntry. It is server-owned
// configuration; it is never supplied or overridden by a client request.
type Catalog map[string]CatalogEntry

// Lookup returns the entry for key, or ErrUnknownPriceKey if key is not in
// the catalog.
func (c Catalog) Lookup(key string) (CatalogEntry, error) {
	entry, ok := c[key]
	if !ok {
		return CatalogEntry{}, ErrUnknownPriceKey
	}
	return entry, nil
}

// ParseCatalogJSON parses a server-owned JSON object mapping price key to
// CatalogEntry (e.g. `{"pro_monthly":{"stripePriceId":"price_123",
// "featureKey":"pro","mode":"subscription"}}`), read from process
// configuration (e.g. BILLING_PRICE_CATALOG) — never from a client request.
// An empty/absent raw value yields an empty Catalog, in which case every
// checkout request is rejected with ErrUnknownPriceKey.
func ParseCatalogJSON(raw string) (Catalog, error) {
	if raw == "" {
		return Catalog{}, nil
	}
	var catalog Catalog
	if err := json.Unmarshal([]byte(raw), &catalog); err != nil {
		return nil, fmt.Errorf("billing: parse price catalog: %w", err)
	}
	for key, entry := range catalog {
		if entry.StripePriceID == "" || entry.FeatureKey == "" {
			return nil, fmt.Errorf("billing: price catalog entry %q is missing stripePriceId or featureKey", key)
		}
		if entry.Mode != ModeSubscription && entry.Mode != ModePayment {
			return nil, fmt.Errorf("billing: price catalog entry %q has invalid mode %q", key, entry.Mode)
		}
	}
	return catalog, nil
}

// FeatureKeyForPrice reverse-looks-up the feature key granted by a Stripe
// price ID, as seen on an incoming subscription webhook event. found is
// false when the price is not in the catalog (e.g. a price created directly
// in the Stripe dashboard outside this service's catalog); callers should
// still project the subscription's status but skip entitlement changes.
func (c Catalog) FeatureKeyForPrice(stripePriceID string) (featureKey string, priceKey string, found bool) {
	for key, entry := range c {
		if entry.StripePriceID == stripePriceID {
			return entry.FeatureKey, key, true
		}
	}
	return "", "", false
}
