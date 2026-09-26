import "@typespec/http";
import "@typespec/openapi";
import "../../common/models.tsp";
import "../../common/responses.tsp";
import "./models.tsp";

using TypeSpec.OpenAPI;
using Starter.Common;

namespace Starter.Billing;

@TypeSpec.Http.route("/api/organizations/{organizationId}/billing")
@tag("Billing")
interface Billing {
  @TypeSpec.Http.route("/summary")
  @TypeSpec.Http.get
  @operationId("getBillingSummary")
  @summary("Get the organization's current billing/entitlement summary")
  getBillingSummary(
    @TypeSpec.Http.path @format("uuid") organizationId: string,
  ):
    | BillingSummary
    | NotFoundResponse
    | UnauthorizedResponse
    | ForbiddenResponse;

  @TypeSpec.Http.route("/checkout-sessions")
  @TypeSpec.Http.post
  @operationId("createCheckoutSession")
  @summary("Create a Stripe Checkout session for a server-controlled price")
  createCheckoutSession(
    @TypeSpec.Http.path @format("uuid") organizationId: string,
    @TypeSpec.Http.body input: CreateCheckoutSessionInput,
  ):
    | CreatedResponse<CheckoutSessionResponse>
    | BadRequestResponse
    | NotFoundResponse
    | UnauthorizedResponse
    | ForbiddenResponse;

  @TypeSpec.Http.route("/portal-sessions")
  @TypeSpec.Http.post
  @operationId("createPortalSession")
  @summary("Create a Stripe Customer Portal session")
  createPortalSession(
    @TypeSpec.Http.path @format("uuid") organizationId: string,
  ):
    | CreatedResponse<PortalSessionResponse>
    | NotFoundResponse
    | UnauthorizedResponse
    | ForbiddenResponse;
}
