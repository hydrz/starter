package billing

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hydrz/starter/internal/platform/database"
	"github.com/hydrz/starter/internal/store"
)

func parseUUID(value string) (pgtype.UUID, error) {
	parsed, err := uuid.Parse(value)
	if err != nil {
		return pgtype.UUID{}, fmt.Errorf("%w: invalid id", ErrInvalidInput)
	}
	return pgtype.UUID{Bytes: parsed, Valid: true}, nil
}

func mustUUIDString(id pgtype.UUID) string {
	return uuid.UUID(id.Bytes).String()
}

func timestamptzFromPointer(value *time.Time) pgtype.Timestamptz {
	if value == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *value, Valid: true}
}

func pointerFromTimestamptz(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	t := value.Time
	return &t
}

func nullableUUIDFromPointer(value *string) (pgtype.UUID, error) {
	if value == nil || *value == "" {
		return pgtype.UUID{}, nil
	}
	return parseUUID(*value)
}

func pointerFromNullableUUID(value pgtype.UUID) *string {
	if !value.Valid {
		return nil
	}
	s := mustUUIDString(value)
	return &s
}

// PostgresBillingAccountRepository implements BillingAccountRepository over internal/store.
type PostgresBillingAccountRepository struct {
	queries *store.Queries
}

func NewPostgresBillingAccountRepository(queries *store.Queries) *PostgresBillingAccountRepository {
	return &PostgresBillingAccountRepository{queries: queries}
}

func (r *PostgresBillingAccountRepository) q(ctx context.Context) *store.Queries {
	if tx, ok := database.TxFromContext(ctx); ok {
		return r.queries.WithTx(tx)
	}
	return r.queries
}

func (r *PostgresBillingAccountRepository) GetOrCreate(ctx context.Context, organizationID, stripeCustomerID string) (BillingAccount, error) {
	orgUUID, err := parseUUID(organizationID)
	if err != nil {
		return BillingAccount{}, err
	}
	row, err := r.q(ctx).CreateBillingAccount(ctx, store.CreateBillingAccountParams{
		OrganizationID:   orgUUID,
		StripeCustomerID: stripeCustomerID,
	})
	if err != nil {
		return BillingAccount{}, fmt.Errorf("get or create billing account: %w", err)
	}
	return billingAccountFromStore(row), nil
}

func (r *PostgresBillingAccountRepository) GetByOrganization(ctx context.Context, organizationID string) (BillingAccount, error) {
	orgUUID, err := parseUUID(organizationID)
	if err != nil {
		return BillingAccount{}, err
	}
	row, err := r.q(ctx).GetBillingAccountByOrganization(ctx, orgUUID)
	if errors.Is(err, pgx.ErrNoRows) {
		return BillingAccount{}, ErrNotFound
	}
	if err != nil {
		return BillingAccount{}, fmt.Errorf("get billing account by organization: %w", err)
	}
	return billingAccountFromStore(row), nil
}

func (r *PostgresBillingAccountRepository) GetByStripeCustomerID(ctx context.Context, stripeCustomerID string) (BillingAccount, error) {
	row, err := r.q(ctx).GetBillingAccountByStripeCustomerID(ctx, stripeCustomerID)
	if errors.Is(err, pgx.ErrNoRows) {
		return BillingAccount{}, ErrNotFound
	}
	if err != nil {
		return BillingAccount{}, fmt.Errorf("get billing account by stripe customer id: %w", err)
	}
	return billingAccountFromStore(row), nil
}

func billingAccountFromStore(row store.BillingAccount) BillingAccount {
	return BillingAccount{
		ID:               mustUUIDString(row.ID),
		OrganizationID:   mustUUIDString(row.OrganizationID),
		StripeCustomerID: row.StripeCustomerID,
		CreatedAt:        row.CreatedAt.Time,
		UpdatedAt:        row.UpdatedAt.Time,
	}
}

// PostgresCheckoutSessionRepository implements CheckoutSessionRepository.
type PostgresCheckoutSessionRepository struct {
	queries *store.Queries
}

func NewPostgresCheckoutSessionRepository(queries *store.Queries) *PostgresCheckoutSessionRepository {
	return &PostgresCheckoutSessionRepository{queries: queries}
}

func (r *PostgresCheckoutSessionRepository) q(ctx context.Context) *store.Queries {
	if tx, ok := database.TxFromContext(ctx); ok {
		return r.queries.WithTx(tx)
	}
	return r.queries
}

func (r *PostgresCheckoutSessionRepository) Create(ctx context.Context, billingAccountID, stripeCheckoutSessionID, priceKey, mode string) (CheckoutSession, error) {
	accountUUID, err := parseUUID(billingAccountID)
	if err != nil {
		return CheckoutSession{}, err
	}
	row, err := r.q(ctx).CreateCheckoutSession(ctx, store.CreateCheckoutSessionParams{
		BillingAccountID:        accountUUID,
		StripeCheckoutSessionID: stripeCheckoutSessionID,
		PriceKey:                priceKey,
		Mode:                    mode,
		Status:                  "open",
	})
	if err != nil {
		return CheckoutSession{}, fmt.Errorf("create checkout session: %w", err)
	}
	return checkoutSessionFromStore(row), nil
}

func (r *PostgresCheckoutSessionRepository) GetByStripeID(ctx context.Context, stripeCheckoutSessionID string) (CheckoutSession, bool, error) {
	row, err := r.q(ctx).GetCheckoutSessionByStripeID(ctx, stripeCheckoutSessionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return CheckoutSession{}, false, nil
	}
	if err != nil {
		return CheckoutSession{}, false, fmt.Errorf("get checkout session: %w", err)
	}
	return checkoutSessionFromStore(row), true, nil
}

func (r *PostgresCheckoutSessionRepository) UpdateStatus(ctx context.Context, stripeCheckoutSessionID, status string) error {
	_, err := r.q(ctx).UpdateCheckoutSessionStatus(ctx, store.UpdateCheckoutSessionStatusParams{
		StripeCheckoutSessionID: stripeCheckoutSessionID,
		Status:                  status,
	})
	if err != nil {
		return fmt.Errorf("update checkout session status: %w", err)
	}
	return nil
}

func checkoutSessionFromStore(row store.CheckoutSession) CheckoutSession {
	return CheckoutSession{
		ID:                      mustUUIDString(row.ID),
		BillingAccountID:        mustUUIDString(row.BillingAccountID),
		StripeCheckoutSessionID: row.StripeCheckoutSessionID,
		PriceKey:                row.PriceKey,
		Mode:                    row.Mode,
		Status:                  row.Status,
		CreatedAt:               row.CreatedAt.Time,
		UpdatedAt:               row.UpdatedAt.Time,
	}
}

// PostgresSubscriptionRepository implements SubscriptionRepository.
type PostgresSubscriptionRepository struct {
	queries *store.Queries
}

func NewPostgresSubscriptionRepository(queries *store.Queries) *PostgresSubscriptionRepository {
	return &PostgresSubscriptionRepository{queries: queries}
}

func (r *PostgresSubscriptionRepository) q(ctx context.Context) *store.Queries {
	if tx, ok := database.TxFromContext(ctx); ok {
		return r.queries.WithTx(tx)
	}
	return r.queries
}

func (r *PostgresSubscriptionRepository) Upsert(ctx context.Context, input UpsertSubscriptionInput) (Subscription, bool, error) {
	accountUUID, err := parseUUID(input.BillingAccountID)
	if err != nil {
		return Subscription{}, false, err
	}
	row, err := r.q(ctx).UpsertSubscription(ctx, store.UpsertSubscriptionParams{
		BillingAccountID:     accountUUID,
		StripeSubscriptionID: input.StripeSubscriptionID,
		PriceKey:             input.PriceKey,
		Status:               input.Status,
		CurrentPeriodEnd:     timestamptzFromPointer(input.CurrentPeriodEnd),
		CancelAtPeriodEnd:    input.CancelAtPeriodEnd,
		LastEventCreatedAt:   pgtype.Timestamptz{Time: input.EventCreatedAt, Valid: true},
	})
	if errors.Is(err, pgx.ErrNoRows) {
		// The monotonic guard rejected a stale/out-of-order event: no-op,
		// not an error.
		return Subscription{}, false, nil
	}
	if err != nil {
		return Subscription{}, false, fmt.Errorf("upsert subscription: %w", err)
	}
	return subscriptionFromStore(row), true, nil
}

func (r *PostgresSubscriptionRepository) GetByStripeID(ctx context.Context, stripeSubscriptionID string) (Subscription, bool, error) {
	row, err := r.q(ctx).GetSubscriptionByStripeID(ctx, stripeSubscriptionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Subscription{}, false, nil
	}
	if err != nil {
		return Subscription{}, false, fmt.Errorf("get subscription: %w", err)
	}
	return subscriptionFromStore(row), true, nil
}

func (r *PostgresSubscriptionRepository) GetLatestForBillingAccount(ctx context.Context, billingAccountID string) (Subscription, bool, error) {
	accountUUID, err := parseUUID(billingAccountID)
	if err != nil {
		return Subscription{}, false, err
	}
	rows, err := r.q(ctx).ListSubscriptionsForBillingAccount(ctx, accountUUID)
	if err != nil {
		return Subscription{}, false, fmt.Errorf("list subscriptions: %w", err)
	}
	if len(rows) == 0 {
		return Subscription{}, false, nil
	}
	return subscriptionFromStore(rows[0]), true, nil
}

func subscriptionFromStore(row store.Subscription) Subscription {
	return Subscription{
		ID:                   mustUUIDString(row.ID),
		BillingAccountID:     mustUUIDString(row.BillingAccountID),
		StripeSubscriptionID: row.StripeSubscriptionID,
		PriceKey:             row.PriceKey,
		Status:               row.Status,
		CurrentPeriodEnd:     pointerFromTimestamptz(row.CurrentPeriodEnd),
		CancelAtPeriodEnd:    row.CancelAtPeriodEnd,
		LastEventCreatedAt:   pointerFromTimestamptz(row.LastEventCreatedAt),
		CreatedAt:            row.CreatedAt.Time,
		UpdatedAt:            row.UpdatedAt.Time,
	}
}

// PostgresOneTimePurchaseRepository implements OneTimePurchaseRepository.
type PostgresOneTimePurchaseRepository struct {
	queries *store.Queries
}

func NewPostgresOneTimePurchaseRepository(queries *store.Queries) *PostgresOneTimePurchaseRepository {
	return &PostgresOneTimePurchaseRepository{queries: queries}
}

func (r *PostgresOneTimePurchaseRepository) q(ctx context.Context) *store.Queries {
	if tx, ok := database.TxFromContext(ctx); ok {
		return r.queries.WithTx(tx)
	}
	return r.queries
}

func (r *PostgresOneTimePurchaseRepository) Insert(ctx context.Context, input InsertOneTimePurchaseInput) (OneTimePurchase, bool, error) {
	accountUUID, err := parseUUID(input.BillingAccountID)
	if err != nil {
		return OneTimePurchase{}, false, err
	}
	row, err := r.q(ctx).InsertOneTimePurchase(ctx, store.InsertOneTimePurchaseParams{
		BillingAccountID:        accountUUID,
		StripeCheckoutSessionID: input.StripeCheckoutSessionID,
		PriceKey:                input.PriceKey,
		AmountMinorUnits:        input.AmountMinorUnits,
		Currency:                input.Currency,
		Status:                  "completed",
	})
	if errors.Is(err, pgx.ErrNoRows) {
		// Already recorded by an earlier delivery: no-op, not an error.
		return OneTimePurchase{}, false, nil
	}
	if err != nil {
		return OneTimePurchase{}, false, fmt.Errorf("insert one-time purchase: %w", err)
	}
	return oneTimePurchaseFromStore(row), true, nil
}

func oneTimePurchaseFromStore(row store.OneTimePurchase) OneTimePurchase {
	return OneTimePurchase{
		ID:                      mustUUIDString(row.ID),
		BillingAccountID:        mustUUIDString(row.BillingAccountID),
		StripeCheckoutSessionID: row.StripeCheckoutSessionID,
		PriceKey:                row.PriceKey,
		AmountMinorUnits:        row.AmountMinorUnits,
		Currency:                row.Currency,
		Status:                  row.Status,
		CreatedAt:               row.CreatedAt.Time,
	}
}

// PostgresWebhookEventRepository implements WebhookEventRepository.
type PostgresWebhookEventRepository struct {
	queries *store.Queries
}

func NewPostgresWebhookEventRepository(queries *store.Queries) *PostgresWebhookEventRepository {
	return &PostgresWebhookEventRepository{queries: queries}
}

func (r *PostgresWebhookEventRepository) q(ctx context.Context) *store.Queries {
	if tx, ok := database.TxFromContext(ctx); ok {
		return r.queries.WithTx(tx)
	}
	return r.queries
}

func (r *PostgresWebhookEventRepository) Insert(ctx context.Context, stripeEventID, eventType string, eventCreatedAt time.Time) (bool, error) {
	_, err := r.q(ctx).InsertWebhookEvent(ctx, store.InsertWebhookEventParams{
		StripeEventID:  stripeEventID,
		EventType:      eventType,
		EventCreatedAt: pgtype.Timestamptz{Time: eventCreatedAt, Valid: true},
	})
	if errors.Is(err, pgx.ErrNoRows) {
		// ON CONFLICT DO NOTHING matched an existing row: duplicate delivery.
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("insert webhook event: %w", err)
	}
	return true, nil
}

// PostgresEntitlementRepository implements EntitlementRepository.
type PostgresEntitlementRepository struct {
	queries *store.Queries
}

func NewPostgresEntitlementRepository(queries *store.Queries) *PostgresEntitlementRepository {
	return &PostgresEntitlementRepository{queries: queries}
}

func (r *PostgresEntitlementRepository) q(ctx context.Context) *store.Queries {
	if tx, ok := database.TxFromContext(ctx); ok {
		return r.queries.WithTx(tx)
	}
	return r.queries
}

func (r *PostgresEntitlementRepository) Upsert(ctx context.Context, input UpsertEntitlementInput) (Entitlement, error) {
	orgUUID, err := parseUUID(input.OrganizationID)
	if err != nil {
		return Entitlement{}, err
	}
	subscriptionUUID, err := nullableUUIDFromPointer(input.SubscriptionID)
	if err != nil {
		return Entitlement{}, err
	}
	purchaseUUID, err := nullableUUIDFromPointer(input.OneTimePurchaseID)
	if err != nil {
		return Entitlement{}, err
	}
	row, err := r.q(ctx).UpsertEntitlement(ctx, store.UpsertEntitlementParams{
		OrganizationID:    orgUUID,
		FeatureKey:        input.FeatureKey,
		Source:            input.Source,
		Enabled:           input.Enabled,
		ExpiresAt:         timestamptzFromPointer(input.ExpiresAt),
		SubscriptionID:    subscriptionUUID,
		OneTimePurchaseID: purchaseUUID,
	})
	if err != nil {
		return Entitlement{}, fmt.Errorf("upsert entitlement: %w", err)
	}
	return entitlementFromStore(row), nil
}

func (r *PostgresEntitlementRepository) HasEntitlement(ctx context.Context, organizationID, featureKey string) (bool, error) {
	orgUUID, err := parseUUID(organizationID)
	if err != nil {
		return false, err
	}
	has, err := r.q(ctx).HasEntitlement(ctx, store.HasEntitlementParams{
		OrganizationID: orgUUID,
		FeatureKey:     featureKey,
	})
	if err != nil {
		return false, fmt.Errorf("has entitlement: %w", err)
	}
	return has, nil
}

func (r *PostgresEntitlementRepository) ListForOrganization(ctx context.Context, organizationID string) ([]Entitlement, error) {
	orgUUID, err := parseUUID(organizationID)
	if err != nil {
		return nil, err
	}
	rows, err := r.q(ctx).ListEntitlementsForOrganization(ctx, orgUUID)
	if err != nil {
		return nil, fmt.Errorf("list entitlements: %w", err)
	}
	entitlements := make([]Entitlement, 0, len(rows))
	for _, row := range rows {
		entitlements = append(entitlements, entitlementFromStore(row))
	}
	return entitlements, nil
}

func entitlementFromStore(row store.Entitlement) Entitlement {
	return Entitlement{
		ID:                mustUUIDString(row.ID),
		OrganizationID:    mustUUIDString(row.OrganizationID),
		FeatureKey:        row.FeatureKey,
		Source:            row.Source,
		Enabled:           row.Enabled,
		ExpiresAt:         pointerFromTimestamptz(row.ExpiresAt),
		SubscriptionID:    pointerFromNullableUUID(row.SubscriptionID),
		OneTimePurchaseID: pointerFromNullableUUID(row.OneTimePurchaseID),
		CreatedAt:         row.CreatedAt.Time,
		UpdatedAt:         row.UpdatedAt.Time,
	}
}

// PostgresOutboxWriter implements OutboxWriter by inserting a row into
// outbox_events. It never performs delivery itself (ADR-0005/0007); a
// separate worker in internal/delivery claims and processes these rows.
type PostgresOutboxWriter struct {
	queries *store.Queries
}

func NewPostgresOutboxWriter(queries *store.Queries) *PostgresOutboxWriter {
	return &PostgresOutboxWriter{queries: queries}
}

func (w *PostgresOutboxWriter) q(ctx context.Context) *store.Queries {
	if tx, ok := database.TxFromContext(ctx); ok {
		return w.queries.WithTx(tx)
	}
	return w.queries
}

func (w *PostgresOutboxWriter) WriteEvent(ctx context.Context, topic, aggregateType, aggregateID string, payload []byte, idempotencyKey string) error {
	parsedAggregate, err := parseUUID(aggregateID)
	if err != nil {
		return fmt.Errorf("invalid aggregate id: %w", err)
	}
	_, err = w.q(ctx).CreateOutboxEventIfAbsent(ctx, store.CreateOutboxEventIfAbsentParams{
		Topic:          topic,
		AggregateType:  aggregateType,
		AggregateID:    parsedAggregate,
		Payload:        payload,
		IdempotencyKey: idempotencyKey,
		AvailableAt:    pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
	})
	if errors.Is(err, pgx.ErrNoRows) {
		// ON CONFLICT DO NOTHING rejected a duplicate idempotency key:
		// already queued, not an error. Unlike catching a raised
		// unique-violation error, this never leaves an enclosing
		// transaction aborted (see the query's comment in
		// db/queries/billing.sql).
		return nil
	}
	return err
}
