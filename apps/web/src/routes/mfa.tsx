import { zodResolver } from "@hookform/resolvers/zod";
import { Link, createFileRoute, useNavigate } from "@tanstack/react-router";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { useVerifyMFAChallenge } from "../api/generated/auth/auth";
import { Alert, AlertDescription } from "../components/ui/alert";
import { Button } from "../components/ui/button";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "../components/ui/form";
import { Input } from "../components/ui/input";
import { AuthLayout } from "../features/auth/AuthLayout";
import { applyAuthApiError } from "../features/auth/form-error";
import { setIssuedTokens } from "../lib/auth-session";
import * as m from "../paraglide/messages";

const searchSchema = z.object({
  challengeId: z.string().optional(),
});

export const Route = createFileRoute("/mfa")({
  validateSearch: searchSchema,
  component: MfaPage,
});

function buildSchema() {
  return z.object({
    code: z.string().min(1, m.auth_validation_required()),
  });
}
type MfaForm = z.infer<ReturnType<typeof buildSchema>>;

function MfaPage() {
  const { challengeId } = Route.useSearch();
  const navigate = useNavigate();
  const form = useForm<MfaForm>({
    resolver: zodResolver(buildSchema()),
    defaultValues: { code: "" },
  });

  const verifyChallenge = useVerifyMFAChallenge({
    mutation: {
      onSuccess: (res) => {
        if (res.status === 200) {
          setIssuedTokens(res.data);
          void navigate({ to: "/app" });
        }
      },
      onError: (error) => applyAuthApiError(form, error),
    },
  });

  if (!challengeId) {
    return (
      <AuthLayout title={m.auth_mfa_title()}>
        <Alert variant="destructive">
          <AlertDescription>{m.auth_mfa_missing_challenge()}</AlertDescription>
        </Alert>
        <Link to="/sign-in">
          <Button className="mt-5 w-full">
            {m.auth_mfa_back_to_sign_in()}
          </Button>
        </Link>
      </AuthLayout>
    );
  }

  return (
    <AuthLayout title={m.auth_mfa_title()} subtitle={m.auth_mfa_subtitle()}>
      <Form {...form}>
        <form
          onSubmit={form.handleSubmit((values) =>
            verifyChallenge.mutate({
              data: { challengeId, code: values.code },
            }),
          )}
          className="grid gap-4"
          noValidate
        >
          {form.formState.errors.root ? (
            <Alert variant="destructive">
              <AlertDescription>
                {form.formState.errors.root.message}
              </AlertDescription>
            </Alert>
          ) : null}

          <FormField
            control={form.control}
            name="code"
            render={({ field }) => (
              <FormItem>
                <FormLabel>{m.auth_mfa_field_code()}</FormLabel>
                <FormControl>
                  <Input
                    inputMode="numeric"
                    autoComplete="one-time-code"
                    autoFocus
                    {...field}
                  />
                </FormControl>
                <p className="text-xs text-muted-foreground">
                  {m.auth_mfa_use_recovery_hint()}
                </p>
                <FormMessage />
              </FormItem>
            )}
          />

          <Button
            type="submit"
            className="w-full"
            disabled={verifyChallenge.isPending}
          >
            {verifyChallenge.isPending
              ? m.auth_mfa_submitting()
              : m.auth_mfa_submit()}
          </Button>
        </form>
      </Form>
    </AuthLayout>
  );
}
