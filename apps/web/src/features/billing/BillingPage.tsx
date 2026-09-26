import { AlertTriangle, CreditCard, ExternalLink } from "lucide-react";
import { useState } from "react";
import {
  useCreateCheckoutSession,
  useCreatePortalSession,
  useGetBillingSummary,
} from "../../api/generated/billing/billing";
import { Alert, AlertDescription } from "../../components/ui/alert";
import { Badge } from "../../components/ui/badge";
import { Button } from "../../components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "../../components/ui/card";
import { Skeleton } from "../../components/ui/skeleton";
import { toast } from "../../components/ui/toast";
import { useOrgContext } from "../organizations/OrgContext";
import { getErrorMessage } from "../../lib/errors";
import * as m from "../../paraglide/messages";
import { PLAN_CATALOG } from "./plan-catalog";

/**
 * `/app/$orgSlug/billing`（DESIGN.md §6.2）：entitlement/订阅状态、
 * 发起 Checkout、Customer Portal 入口。
 *
 * Checkout/Portal 都返回一个 Stripe 托管页面的 URL——拿到后整页跳转过去
 * （`window.location.assign`），不在应用内嵌入 iframe，这与后端把
 * Checkout/Portal 会话当作"一次性、服务端签发的外部 URL"的模型一致。
 */
export function BillingPage() {
  const { organizationId } = useOrgContext();
  const summaryQuery = useGetBillingSummary(organizationId);
  const summary =
    summaryQuery.data?.status === 200 ? summaryQuery.data.data : undefined;

  const [checkoutPending, setCheckoutPending] = useState<string | null>(null);

  const createCheckoutSession = useCreateCheckoutSession({
    mutation: {
      onSuccess: (res) => {
        if (res.status === 201) {
          window.location.assign(res.data.url);
          return;
        }
        setCheckoutPending(null);
      },
      onError: (error) => {
        toast.error(getErrorMessage(error));
        setCheckoutPending(null);
      },
    },
  });

  const createPortalSession = useCreatePortalSession({
    mutation: {
      onSuccess: (res) => {
        if (res.status === 201) {
          window.location.assign(res.data.url);
        }
      },
      onError: (error) => toast.error(getErrorMessage(error)),
    },
  });

  if (summaryQuery.isPending) {
    return (
      <div className="max-w-2xl space-y-3">
        <Skeleton className="h-24 w-full" />
        <Skeleton className="h-24 w-full" />
      </div>
    );
  }

  if (summaryQuery.isError || !summary) {
    return (
      <Alert variant="destructive">
        <AlertTriangle />
        <AlertDescription>
          {summaryQuery.error
            ? getErrorMessage(summaryQuery.error)
            : m.billing_load_error()}
        </AlertDescription>
      </Alert>
    );
  }

  return (
    <div className="max-w-2xl space-y-6">
      {summary.hasActiveSubscription ? (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-base">
              <CreditCard size={16} />
              {m.billing_active_subscription_heading()}
            </CardTitle>
            {summary.subscription && (
              <CardDescription>
                {m.billing_subscription_status({
                  status: summary.subscription.status,
                })}
                {summary.subscription.cancelAtPeriodEnd &&
                  ` · ${m.billing_cancel_at_period_end()}`}
              </CardDescription>
            )}
          </CardHeader>
          <CardContent className="space-y-4">
            {summary.entitlements.length > 0 && (
              <ul className="space-y-1.5">
                {summary.entitlements.map((entitlement) => (
                  <li
                    key={entitlement.featureKey}
                    className="flex items-center justify-between gap-3 rounded-md border border-border px-3 py-2"
                  >
                    <span className="font-mono text-xs">
                      {entitlement.featureKey}
                    </span>
                    <Badge
                      variant={entitlement.enabled ? "default" : "outline"}
                    >
                      {entitlement.enabled
                        ? m.billing_entitlement_enabled()
                        : m.billing_entitlement_disabled()}
                    </Badge>
                  </li>
                ))}
              </ul>
            )}

            <Button
              type="button"
              variant="outline"
              onClick={() => createPortalSession.mutate({ organizationId })}
              disabled={createPortalSession.isPending}
            >
              {createPortalSession.isPending
                ? m.billing_portal_pending()
                : m.billing_portal_action()}
              <ExternalLink size={14} />
            </Button>
          </CardContent>
        </Card>
      ) : (
        <Alert>
          <AlertDescription>{m.billing_no_subscription()}</AlertDescription>
        </Alert>
      )}

      {!summary.hasActiveSubscription && (
        <section className="grid gap-4 sm:grid-cols-2">
          {PLAN_CATALOG.map((plan) => (
            <Card key={plan.priceKey}>
              <CardHeader>
                <CardTitle className="text-base">{plan.name}</CardTitle>
                <CardDescription>{plan.description}</CardDescription>
              </CardHeader>
              <CardContent className="space-y-3">
                <p className="font-mono text-xl font-semibold text-foreground">
                  {plan.priceDisplay}
                </p>
                <Button
                  type="button"
                  className="w-full"
                  disabled={
                    createCheckoutSession.isPending &&
                    checkoutPending === plan.priceKey
                  }
                  onClick={() => {
                    setCheckoutPending(plan.priceKey);
                    createCheckoutSession.mutate({
                      organizationId,
                      data: { priceKey: plan.priceKey },
                    });
                  }}
                >
                  {createCheckoutSession.isPending &&
                  checkoutPending === plan.priceKey
                    ? m.billing_checkout_pending()
                    : m.billing_checkout_action()}
                </Button>
              </CardContent>
            </Card>
          ))}
        </section>
      )}
    </div>
  );
}
