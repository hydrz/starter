import { Link } from "@tanstack/react-router";
import { AlertTriangle, ArrowRight, KeyRound, ShieldCheck } from "lucide-react";
import { useGetCurrentIdentity } from "../../api/generated/auth/auth";
import { Alert, AlertDescription } from "../../components/ui/alert";
import { Badge } from "../../components/ui/badge";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../../components/ui/card";
import { Code } from "../../components/ui/code";
import { Skeleton } from "../../components/ui/skeleton";
import { getErrorMessage } from "../../lib/errors";
import * as m from "../../paraglide/messages";

/**
 * `/app/account`（DESIGN.md §6.2）：账户资料概览。
 *
 * 密码修改：本后端没有"已登录状态下修改密码"接口（只有
 * `/auth/password-reset/*` 邮箱令牌流程，Stage 2 已经在
 * `/forgot-password`/`/reset-password` 实现），这里不臆造一个不存在的
 * 表单，只链接到那个已有流程。
 */
export function AccountOverviewPage() {
  const identityQuery = useGetCurrentIdentity();
  const identity =
    identityQuery.data?.status === 200 ? identityQuery.data.data : undefined;

  return (
    <div className="max-w-lg space-y-6">
      <Card>
        <CardHeader>
          <CardTitle className="text-base">
            {m.account_overview_profile_heading()}
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          {identityQuery.isPending && (
            <div className="space-y-2">
              <Skeleton className="h-5 w-48" />
              <Skeleton className="h-5 w-24" />
            </div>
          )}

          {identityQuery.isError && (
            <Alert variant="destructive">
              <AlertTriangle />
              <AlertDescription>
                {getErrorMessage(identityQuery.error)}
              </AlertDescription>
            </Alert>
          )}

          {identity && (
            <div className="space-y-2">
              <div className="flex items-center justify-between gap-3">
                <span className="text-xs font-medium text-muted-foreground">
                  {m.account_overview_email_label()}
                </span>
                <Code copyable={false} className="text-xs">
                  {identity.email}
                </Code>
              </div>
              <div className="flex items-center justify-between gap-3">
                <span className="text-xs font-medium text-muted-foreground">
                  {m.account_overview_email_verified_label()}
                </span>
                <Badge variant={identity.emailVerified ? "default" : "outline"}>
                  {identity.emailVerified
                    ? m.account_overview_email_verified_yes()
                    : m.account_overview_email_verified_no()}
                </Badge>
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">
            {m.account_overview_password_heading()}
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <p className="text-sm text-muted-foreground">
            {m.account_overview_password_description()}
          </p>
          <Link to="/forgot-password">
            <span className="inline-flex items-center gap-1 text-sm font-medium text-primary hover:underline">
              {m.account_overview_password_action()}
              <ArrowRight size={14} />
            </span>
          </Link>
        </CardContent>
      </Card>

      <div className="grid gap-4 sm:grid-cols-2">
        <Link to="/app/account/sessions">
          <Card className="h-full transition-colors hover:border-primary/50">
            <CardHeader className="flex flex-row items-center gap-2 space-y-0">
              <KeyRound size={16} className="text-muted-foreground" />
              <CardTitle className="text-sm">
                {m.account_overview_sessions_link()}
              </CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-xs text-muted-foreground">
                {m.account_overview_sessions_link_description()}
              </p>
            </CardContent>
          </Card>
        </Link>
        <Link to="/app/account/security">
          <Card className="h-full transition-colors hover:border-primary/50">
            <CardHeader className="flex flex-row items-center gap-2 space-y-0">
              <ShieldCheck size={16} className="text-muted-foreground" />
              <CardTitle className="text-sm">
                {m.account_overview_security_link()}
              </CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-xs text-muted-foreground">
                {m.account_overview_security_link_description()}
              </p>
            </CardContent>
          </Card>
        </Link>
      </div>
    </div>
  );
}
