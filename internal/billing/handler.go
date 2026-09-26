package billing

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/hydrz/starter/internal/api/billingapi"
)

// HTTPHandler adapts billingapi.Handler (generated from
// packages/contracts/features/billing) to Service. It never leaks store/
// ogen or Stripe SDK types outside this file.
//
// Every route is mounted (see internal/platform/httpserver) behind, in
// order: authentication middleware, organization-membership + Casbin
// RequirePermission middleware, and then this handler, which always
// resolves the billing account through the URL's organizationId — it never
// trusts a client-supplied billing account/customer/subscription ID — so an
// organization can only ever see or act on its own billing account.
type HTTPHandler struct {
	service *Service
}

func NewHTTPHandler(service *Service) *HTTPHandler {
	return &HTTPHandler{service: service}
}

var _ billingapi.Handler = (*HTTPHandler)(nil)

func (h *HTTPHandler) GetBillingSummary(ctx context.Context, params billingapi.GetBillingSummaryParams) (billingapi.GetBillingSummaryRes, error) {
	summary, err := h.service.GetSummary(ctx, params.OrganizationId.String())
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return &billingapi.GetBillingSummaryNotFound{Code: "not_found", Message: "Organization not found."}, nil
		}
		return nil, err
	}
	res := toAPISummary(summary)
	return &res, nil
}

func (h *HTTPHandler) CreateCheckoutSession(ctx context.Context, req *billingapi.CreateCheckoutSessionInput, params billingapi.CreateCheckoutSessionParams) (billingapi.CreateCheckoutSessionRes, error) {
	created, err := h.service.CreateCheckoutSession(ctx, params.OrganizationId.String(), req.PriceKey)
	if err != nil {
		switch {
		case errors.Is(err, ErrUnknownPriceKey), errors.Is(err, ErrInvalidInput):
			return &billingapi.CreateCheckoutSessionBadRequest{Code: "invalid_request", Message: "Unknown or invalid price key."}, nil
		case errors.Is(err, ErrNotFound):
			return &billingapi.CreateCheckoutSessionNotFound{Code: "not_found", Message: "Organization not found."}, nil
		default:
			return nil, err
		}
	}
	return &billingapi.CheckoutSessionResponse{
		ID:  uuid.MustParse(created.ID),
		URL: created.URL,
	}, nil
}

func (h *HTTPHandler) CreatePortalSession(ctx context.Context, params billingapi.CreatePortalSessionParams) (billingapi.CreatePortalSessionRes, error) {
	created, err := h.service.CreatePortalSession(ctx, params.OrganizationId.String())
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return &billingapi.CreatePortalSessionNotFound{Code: "not_found", Message: "This organization has no billing account yet."}, nil
		}
		return nil, err
	}
	return &billingapi.PortalSessionResponse{URL: created.URL}, nil
}

func toAPISummary(summary Summary) billingapi.BillingSummary {
	result := billingapi.BillingSummary{
		OrganizationId:        uuid.MustParse(summary.OrganizationID),
		HasActiveSubscription: summary.HasActiveSubscription,
		Entitlements:          make([]billingapi.EntitlementSummary, 0, len(summary.Entitlements)),
	}
	for _, entitlement := range summary.Entitlements {
		item := billingapi.EntitlementSummary{
			FeatureKey: entitlement.FeatureKey,
			Enabled:    entitlement.Enabled,
		}
		if entitlement.ExpiresAt != nil {
			item.ExpiresAt = billingapi.NewOptDateTime(*entitlement.ExpiresAt)
		}
		result.Entitlements = append(result.Entitlements, item)
	}
	if summary.Subscription != nil {
		sub := billingapi.SubscriptionSummary{
			ID:                uuid.MustParse(summary.Subscription.ID),
			PriceKey:          summary.Subscription.PriceKey,
			Status:            billingapi.SubscriptionStatus(summary.Subscription.Status),
			CancelAtPeriodEnd: summary.Subscription.CancelAtPeriodEnd,
		}
		if summary.Subscription.CurrentPeriodEnd != nil {
			sub.CurrentPeriodEnd = billingapi.NewOptDateTime(*summary.Subscription.CurrentPeriodEnd)
		}
		result.Subscription = billingapi.NewOptSubscriptionSummary(sub)
	}
	return result
}
