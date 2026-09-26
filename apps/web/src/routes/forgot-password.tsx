import { zodResolver } from "@hookform/resolvers/zod";
import { Link, createFileRoute } from "@tanstack/react-router";
import { MailCheck } from "lucide-react";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { useRequestPasswordReset } from "../api/generated/auth/auth";
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
import * as m from "../paraglide/messages";

export const Route = createFileRoute("/forgot-password")({
  component: ForgotPasswordPage,
});

function buildSchema() {
  return z.object({
    email: z.email(m.auth_validation_email()),
  });
}

type ForgotPasswordForm = z.infer<ReturnType<typeof buildSchema>>;

function ForgotPasswordPage() {
  // 反枚举（anti-enumeration）：无论邮箱是否存在于系统中，都展示同一个
  // 成功态，与后端语义保持一致（DESIGN.md 未直接列出，但任务描述明确
  // 要求不得泄露账户存在性）。
  const [requested, setRequested] = useState(false);
  const schema = buildSchema();
  const form = useForm<ForgotPasswordForm>({
    resolver: zodResolver(schema),
    defaultValues: { email: "" },
  });

  const requestReset = useRequestPasswordReset({
    mutation: {
      onSuccess: () => setRequested(true),
      onError: (error) => applyAuthApiError(form, error),
    },
  });

  function onSubmit(values: ForgotPasswordForm) {
    requestReset.mutate({ data: values });
  }

  if (requested) {
    return (
      <AuthLayout title={m.auth_forgot_password_success_title()}>
        <Alert variant="success">
          <MailCheck />
          <AlertDescription>
            {m.auth_forgot_password_success_body()}
          </AlertDescription>
        </Alert>
        <Link to="/sign-in">
          <Button className="mt-5 w-full">
            {m.auth_forgot_password_back_to_sign_in()}
          </Button>
        </Link>
      </AuthLayout>
    );
  }

  return (
    <AuthLayout
      title={m.auth_forgot_password_title()}
      subtitle={m.auth_forgot_password_subtitle()}
      footer={
        <Link
          to="/sign-in"
          className="font-medium text-foreground hover:underline"
        >
          {m.auth_forgot_password_back_to_sign_in()}
        </Link>
      }
    >
      <Form {...form}>
        <form
          onSubmit={form.handleSubmit(onSubmit)}
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
            disabled={requestReset.isPending}
          >
            {requestReset.isPending
              ? m.auth_forgot_password_submitting()
              : m.auth_forgot_password_submit()}
          </Button>
        </form>
      </Form>
    </AuthLayout>
  );
}
