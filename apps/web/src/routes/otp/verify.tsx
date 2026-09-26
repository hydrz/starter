import { zodResolver } from "@hookform/resolvers/zod";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import {
  useRequestEmailOTP,
  useVerifyEmailOTP,
} from "../../api/generated/auth/auth";
import { Alert, AlertDescription } from "../../components/ui/alert";
import { Button } from "../../components/ui/button";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "../../components/ui/form";
import { Input } from "../../components/ui/input";
import { AuthLayout } from "../../features/auth/AuthLayout";
import { applyAuthApiError } from "../../features/auth/form-error";
import { setIssuedTokens } from "../../lib/auth-session";
import * as m from "../../paraglide/messages";

export const Route = createFileRoute("/otp/verify")({
  component: OtpVerifyPage,
});

function emailSchema() {
  return z.object({ email: z.email(m.auth_validation_email()) });
}
type EmailForm = z.infer<ReturnType<typeof emailSchema>>;

function codeSchema() {
  return z.object({
    code: z
      .string()
      .length(6, m.auth_validation_code_length())
      .regex(/^\d{6}$/, m.auth_validation_code_length()),
  });
}
type CodeForm = z.infer<ReturnType<typeof codeSchema>>;

function RequestStep({
  onRequested,
}: {
  onRequested: (email: string) => void;
}) {
  const form = useForm<EmailForm>({
    resolver: zodResolver(emailSchema()),
    defaultValues: { email: "" },
  });

  const requestOtp = useRequestEmailOTP({
    mutation: {
      onSuccess: (_res, variables) => onRequested(variables.data.email),
      onError: (error) => applyAuthApiError(form, error),
    },
  });

  return (
    <AuthLayout title={m.auth_otp_title()} subtitle={m.auth_otp_subtitle()}>
      <Form {...form}>
        <form
          onSubmit={form.handleSubmit((values) =>
            requestOtp.mutate({ data: values }),
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
            name="email"
            render={({ field }) => (
              <FormItem>
                <FormLabel>{m.auth_field_email()}</FormLabel>
                <FormControl>
                  <Input
                    type="email"
                    autoComplete="email"
                    autoFocus
                    {...field}
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />

          <Button
            type="submit"
            className="w-full"
            disabled={requestOtp.isPending}
          >
            {requestOtp.isPending
              ? m.auth_otp_request_submitting()
              : m.auth_otp_request_submit()}
          </Button>
        </form>
      </Form>
    </AuthLayout>
  );
}

function VerifyStep({
  email,
  onChangeEmail,
}: {
  email: string;
  onChangeEmail: () => void;
}) {
  const navigate = useNavigate();
  const form = useForm<CodeForm>({
    resolver: zodResolver(codeSchema()),
    defaultValues: { code: "" },
  });

  const requestOtp = useRequestEmailOTP();

  const verifyOtp = useVerifyEmailOTP({
    mutation: {
      onSuccess: (res) => {
        if (res.status === 200) {
          setIssuedTokens(res.data);
          void navigate({ to: "/app" });
          return;
        }
        if (res.status === 202) {
          void navigate({
            to: "/mfa",
            search: { challengeId: res.data.challengeId },
          });
        }
      },
      onError: (error) => applyAuthApiError(form, error),
    },
  });

  return (
    <AuthLayout
      title={m.auth_otp_verify_title()}
      subtitle={m.auth_otp_verify_subtitle({ email })}
      footer={
        <button
          type="button"
          onClick={onChangeEmail}
          className="font-medium text-foreground hover:underline"
        >
          {m.auth_otp_change_email()}
        </button>
      }
    >
      <Form {...form}>
        <form
          onSubmit={form.handleSubmit((values) =>
            verifyOtp.mutate({ data: { email, code: values.code } }),
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
                <FormLabel>{m.auth_field_code()}</FormLabel>
                <FormControl>
                  <Input
                    inputMode="numeric"
                    maxLength={6}
                    autoComplete="one-time-code"
                    autoFocus
                    {...field}
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />

          <Button
            type="submit"
            className="w-full"
            disabled={verifyOtp.isPending}
          >
            {verifyOtp.isPending
              ? m.auth_otp_verify_submitting()
              : m.auth_otp_verify_submit()}
          </Button>

          <Button
            type="button"
            variant="ghost"
            className="w-full"
            disabled={requestOtp.isPending}
            onClick={() => requestOtp.mutate({ data: { email } })}
          >
            {m.auth_otp_resend()}
          </Button>
        </form>
      </Form>
    </AuthLayout>
  );
}

function OtpVerifyPage() {
  const [email, setEmail] = useState<string | null>(null);

  if (!email) {
    return <RequestStep onRequested={setEmail} />;
  }

  return <VerifyStep email={email} onChangeEmail={() => setEmail(null)} />;
}
