package billing_test

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hydrz/starter/internal/billing"
)

// --- Clock ---------------------------------------------------------------

type fakeClock struct{ now time.Time }

func (c *fakeClock) Now() time.Time { return c.now }

// --- Transactor / Outbox --------------------------------------------------

type fakeTransactor struct{}

func (fakeTransactor) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

type outboxRecord struct {
	topic, aggregateType, aggregateID, idempotencyKey string
	payload                                           []byte
}

type fakeOutbox struct {
	mu     sync.Mutex
	events []outboxRecord
}

func (o *fakeOutbox) WriteEvent(_ context.Context, topic, aggregateType, aggregateID string, payload []byte, idempotencyKey string) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.events = append(o.events, outboxRecord{topic, aggregateType, aggregateID, idempotencyKey, payload})
	return nil
}

// --- Stripe gateway fake --------------------------------------------------

type fakeGateway struct {
	mu               sync.Mutex
	nextCustomer     int
	nextCheckout     int
	createCustomerFn func(organizationID string) (string, error)
	portalCalls      []struct{ customerID, returnURL string }
}

func (g *fakeGateway) CreateCustomer(_ context.Context, organizationID string) (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.createCustomerFn != nil {
		return g.createCustomerFn(organizationID)
	}
	g.nextCustomer++
	return fmt.Sprintf("cus_%d", g.nextCustomer), nil
}

func (g *fakeGateway) CreateCheckoutSession(_ context.Context, params billing.CheckoutParams) (string, string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.nextCheckout++
	id := fmt.Sprintf("cs_%d", g.nextCheckout)
	return id, "https://stripe.test/checkout/" + id, nil
}

func (g *fakeGateway) CreatePortalSession(_ context.Context, customerID, returnURL string) (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.portalCalls = append(g.portalCalls, struct{ customerID, returnURL string }{customerID, returnURL})
	return "https://stripe.test/portal/" + customerID, nil
}

// --- BillingAccountRepository ---------------------------------------------

type memoryBillingAccounts struct {
	mu         sync.Mutex
	byOrg      map[string]billing.BillingAccount
	byCustomer map[string]billing.BillingAccount
	nextID     int
}

func newMemoryBillingAccounts() *memoryBillingAccounts {
	return &memoryBillingAccounts{byOrg: map[string]billing.BillingAccount{}, byCustomer: map[string]billing.BillingAccount{}}
}

func (m *memoryBillingAccounts) GetOrCreate(_ context.Context, organizationID, stripeCustomerID string) (billing.BillingAccount, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if acc, ok := m.byOrg[organizationID]; ok {
		return acc, nil
	}
	m.nextID++
	acc := billing.BillingAccount{
		ID: fmt.Sprintf("ba-%d", m.nextID), OrganizationID: organizationID, StripeCustomerID: stripeCustomerID,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	m.byOrg[organizationID] = acc
	m.byCustomer[stripeCustomerID] = acc
	return acc, nil
}

func (m *memoryBillingAccounts) GetByOrganization(_ context.Context, organizationID string) (billing.BillingAccount, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	acc, ok := m.byOrg[organizationID]
	if !ok {
		return billing.BillingAccount{}, billing.ErrNotFound
	}
	return acc, nil
}

func (m *memoryBillingAccounts) GetByStripeCustomerID(_ context.Context, stripeCustomerID string) (billing.BillingAccount, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	acc, ok := m.byCustomer[stripeCustomerID]
	if !ok {
		return billing.BillingAccount{}, billing.ErrNotFound
	}
	return acc, nil
}

// --- CheckoutSessionRepository ---------------------------------------------

type memoryCheckoutSessions struct {
	mu         sync.Mutex
	byStripeID map[string]billing.CheckoutSession
	nextID     int
}

func newMemoryCheckoutSessions() *memoryCheckoutSessions {
	return &memoryCheckoutSessions{byStripeID: map[string]billing.CheckoutSession{}}
}

func (m *memoryCheckoutSessions) Create(_ context.Context, billingAccountID, stripeCheckoutSessionID, priceKey, mode string) (billing.CheckoutSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextID++
	cs := billing.CheckoutSession{
		ID: fmt.Sprintf("cs-row-%d", m.nextID), BillingAccountID: billingAccountID,
		StripeCheckoutSessionID: stripeCheckoutSessionID, PriceKey: priceKey, Mode: mode, Status: "open",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	m.byStripeID[stripeCheckoutSessionID] = cs
	return cs, nil
}

func (m *memoryCheckoutSessions) GetByStripeID(_ context.Context, stripeCheckoutSessionID string) (billing.CheckoutSession, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cs, ok := m.byStripeID[stripeCheckoutSessionID]
	return cs, ok, nil
}

func (m *memoryCheckoutSessions) UpdateStatus(_ context.Context, stripeCheckoutSessionID, status string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cs, ok := m.byStripeID[stripeCheckoutSessionID]
	if !ok {
		return nil
	}
	cs.Status = status
	m.byStripeID[stripeCheckoutSessionID] = cs
	return nil
}

// --- SubscriptionRepository --------------------------------------------------

type memorySubscriptions struct {
	mu         sync.Mutex
	byStripeID map[string]billing.Subscription
	nextID     int
}

func newMemorySubscriptions() *memorySubscriptions {
	return &memorySubscriptions{byStripeID: map[string]billing.Subscription{}}
}

func (m *memorySubscriptions) Upsert(_ context.Context, input billing.UpsertSubscriptionInput) (billing.Subscription, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, ok := m.byStripeID[input.StripeSubscriptionID]
	// Monotonic guard mirrors the SQL: apply only when strictly newer than
	// the last event that updated this row.
	if ok && existing.LastEventCreatedAt != nil && !input.EventCreatedAt.After(*existing.LastEventCreatedAt) {
		return billing.Subscription{}, false, nil
	}

	id := existing.ID
	created := existing.CreatedAt
	if !ok {
		m.nextID++
		id = fmt.Sprintf("sub-row-%d", m.nextID)
		created = time.Now()
	}
	eventCreatedAt := input.EventCreatedAt
	sub := billing.Subscription{
		ID: id, BillingAccountID: input.BillingAccountID, StripeSubscriptionID: input.StripeSubscriptionID,
		PriceKey: input.PriceKey, Status: input.Status, CurrentPeriodEnd: input.CurrentPeriodEnd,
		CancelAtPeriodEnd: input.CancelAtPeriodEnd, LastEventCreatedAt: &eventCreatedAt,
		CreatedAt: created, UpdatedAt: time.Now(),
	}
	m.byStripeID[input.StripeSubscriptionID] = sub
	return sub, true, nil
}

func (m *memorySubscriptions) GetByStripeID(_ context.Context, stripeSubscriptionID string) (billing.Subscription, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	sub, ok := m.byStripeID[stripeSubscriptionID]
	return sub, ok, nil
}

func (m *memorySubscriptions) GetLatestForBillingAccount(_ context.Context, billingAccountID string) (billing.Subscription, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var latest billing.Subscription
	found := false
	for _, sub := range m.byStripeID {
		if sub.BillingAccountID != billingAccountID {
			continue
		}
		if !found || sub.CreatedAt.After(latest.CreatedAt) {
			latest = sub
			found = true
		}
	}
	return latest, found, nil
}

// --- OneTimePurchaseRepository ----------------------------------------------

type memoryOneTimePurchases struct {
	mu         sync.Mutex
	byStripeID map[string]billing.OneTimePurchase
	nextID     int
}

func newMemoryOneTimePurchases() *memoryOneTimePurchases {
	return &memoryOneTimePurchases{byStripeID: map[string]billing.OneTimePurchase{}}
}

func (m *memoryOneTimePurchases) Insert(_ context.Context, input billing.InsertOneTimePurchaseInput) (billing.OneTimePurchase, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if existing, ok := m.byStripeID[input.StripeCheckoutSessionID]; ok {
		return existing, false, nil
	}
	m.nextID++
	p := billing.OneTimePurchase{
		ID: fmt.Sprintf("otp-%d", m.nextID), BillingAccountID: input.BillingAccountID,
		StripeCheckoutSessionID: input.StripeCheckoutSessionID, PriceKey: input.PriceKey,
		AmountMinorUnits: input.AmountMinorUnits, Currency: input.Currency, Status: "completed",
		CreatedAt: time.Now(),
	}
	m.byStripeID[input.StripeCheckoutSessionID] = p
	return p, true, nil
}

// --- WebhookEventRepository -------------------------------------------------

type memoryWebhookEvents struct {
	mu   sync.Mutex
	seen map[string]bool
}

func newMemoryWebhookEvents() *memoryWebhookEvents {
	return &memoryWebhookEvents{seen: map[string]bool{}}
}

func (m *memoryWebhookEvents) Insert(_ context.Context, stripeEventID, _ string, _ time.Time) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.seen[stripeEventID] {
		return false, nil
	}
	m.seen[stripeEventID] = true
	return true, nil
}

// --- EntitlementRepository ---------------------------------------------------

type entitlementKey struct{ orgID, featureKey, source string }

type memoryEntitlements struct {
	mu     sync.Mutex
	rows   map[entitlementKey]billing.Entitlement
	nextID int
}

func newMemoryEntitlements() *memoryEntitlements {
	return &memoryEntitlements{rows: map[entitlementKey]billing.Entitlement{}}
}

func (m *memoryEntitlements) Upsert(_ context.Context, input billing.UpsertEntitlementInput) (billing.Entitlement, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := entitlementKey{input.OrganizationID, input.FeatureKey, input.Source}
	existing, ok := m.rows[key]
	id := existing.ID
	created := existing.CreatedAt
	if !ok {
		m.nextID++
		id = fmt.Sprintf("ent-%d", m.nextID)
		created = time.Now()
	}
	e := billing.Entitlement{
		ID: id, OrganizationID: input.OrganizationID, FeatureKey: input.FeatureKey, Source: input.Source,
		Enabled: input.Enabled, ExpiresAt: input.ExpiresAt, SubscriptionID: input.SubscriptionID,
		OneTimePurchaseID: input.OneTimePurchaseID, CreatedAt: created, UpdatedAt: time.Now(),
	}
	m.rows[key] = e
	return e, nil
}

func (m *memoryEntitlements) HasEntitlement(_ context.Context, organizationID, featureKey string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	for key, row := range m.rows {
		if key.orgID != organizationID || key.featureKey != featureKey {
			continue
		}
		if row.Enabled && (row.ExpiresAt == nil || row.ExpiresAt.After(now)) {
			return true, nil
		}
	}
	return false, nil
}

func (m *memoryEntitlements) ListForOrganization(_ context.Context, organizationID string) ([]billing.Entitlement, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []billing.Entitlement
	for key, row := range m.rows {
		if key.orgID == organizationID {
			out = append(out, row)
		}
	}
	return out, nil
}

// --- test harness ----------------------------------------------------------

type harness struct {
	billingAccounts  *memoryBillingAccounts
	checkoutSessions *memoryCheckoutSessions
	subscriptions    *memorySubscriptions
	oneTimePurchases *memoryOneTimePurchases
	webhookEvents    *memoryWebhookEvents
	entitlements     *memoryEntitlements
	gateway          *fakeGateway
	outbox           *fakeOutbox
	service          *billing.Service
}

const testWebhookSecret = "whsec_test_secret"

func newHarness(t interface{ Fatalf(string, ...any) }, catalog billing.Catalog) *harness {
	h := &harness{
		billingAccounts:  newMemoryBillingAccounts(),
		checkoutSessions: newMemoryCheckoutSessions(),
		subscriptions:    newMemorySubscriptions(),
		oneTimePurchases: newMemoryOneTimePurchases(),
		webhookEvents:    newMemoryWebhookEvents(),
		entitlements:     newMemoryEntitlements(),
		gateway:          &fakeGateway{},
		outbox:           &fakeOutbox{},
	}
	service, err := billing.NewService(billing.Dependencies{
		BillingAccounts:  h.billingAccounts,
		CheckoutSessions: h.checkoutSessions,
		Subscriptions:    h.subscriptions,
		OneTimePurchases: h.oneTimePurchases,
		WebhookEvents:    h.webhookEvents,
		Entitlements:     h.entitlements,
		Transactor:       fakeTransactor{},
		Outbox:           h.outbox,
		Gateway:          h.gateway,
		Catalog:          catalog,
		Clock:            &fakeClock{now: time.Now()},
		WebhookSecret:    testWebhookSecret,
		SuccessURL:       "https://app.test/billing/success",
		CancelURL:        "https://app.test/billing/cancel",
		PortalReturnURL:  "https://app.test/billing",
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	h.service = service
	return h
}
