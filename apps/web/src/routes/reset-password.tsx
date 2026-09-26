import { zodResolver } from "@hookform/resolvers/zod";
import { Link, createFileRoute, useNavigate } from "@tanstack/react-router";
import { CheckCircle2 } from "lucide-react";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { useConfirmPasswordReset } from "../api/generated/auth/auth";
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

const searchSchema = z.object({
  token: z.string().optional(),
});

export const Route = createFileRoute("/reset-password")({
  validateSearch: searchSchema,
  component: ResetPasswordPage,
});

function buildSchema() {
  return z
    .object({
      password: z
        .string()
        .min(8, m.auth_validation_password_length())
        .max(128, m.auth_validation_password_length()),
      confirmPassword: z.string().min(1, m.auth_validation_required()),
    })
    .refine((values) => values.password === values.confirmPassword, {
      message: m.auth_validation_password_mismatch(),
      path: ["confirmPassword"],
    });
}

type ResetPasswordForm = z.infer<ReturnType<typeof buildSchema>>;

function ResetPasswordPage() {
  const { token } = Route.useSearch();
  const navigate = useNavigate();
  const [succeeded, setSucceeded] = useState(false);
  const schema = buildSchema();
  const form = useForm<ResetPasswordForm>({
    resolver: zodResolver(schema),
    defaultValues: { password: "", confirmPassword: "" },
  });

  const confirmReset = useConfirmPasswordReset({
    mutation: {
      onSuccess: () => setSucceeded(true),
      onError: (error) => applyAuthApiError(form, error),
    },
  });

  function onSubmit(values: ResetPasswordForm) {
    if (!token) {
      return;
    }
    confirmReset.mutate({ data: { token, password: values.password } });
  }

  if (!token) {
    return (
      <AuthLayout title={m.auth_reset_password_title()}>
        <Alert variant="destructive">
          <AlertDescription>
            {m.auth_reset_password_missing_token()}
          </AlertDescription>
        </Alert>
        <Link to="/forgot-password">
          <Button className="mt-5 w-full">
            {m.auth_reset_password_request_new()}
          </Button>
        </Link>
      </AuthLayout>
    );
  }

  if (succeeded) {
    return (
      <AuthLayout title={m.auth_reset_password_success_title()}>
        <Alert variant="success">
          <CheckCircle2 />
          <AlertDescription>
            {m.auth_reset_password_success_body()}
          </AlertDescription>
        </Alert>
        <Button
          className="mt-5 w-full"
          onClick={() => void navigate({ to: "/sign-in" })}
        >
          {m.auth_reset_password_success_action()}
        </Button>
      </AuthLayout>
    );
  }

  return (
    <AuthLayout
      title={m.auth_reset_password_title()}
      subtitle={m.auth_reset_password_subtitle()}
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
            name="password"
            render={({ field }) => (
              <FormItem>
                <FormLabel>{m.auth_field_new_password()}</FormLabel>
                <FormControl>
                  <Input
                    type="password"
                    autoComplete="new-password"
                    autoFocus
                    {...field}
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name="confirmPassword"
            render={({ field }) => (
              <FormItem>
                <FormLabel>{m.auth_field_confirm_password()}</FormLabel>
                <FormControl>
                  <Input
                    type="password"
                    autoComplete="new-password"
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
            disabled={confirmReset.isPending}
          >
            {confirmReset.isPending
              ? m.auth_reset_password_submitting()
              : m.auth_reset_password_submit()}
          </Button>
        </form>
      </Form>
    </AuthLayout>
  );
}
