import { zodResolver } from "@hookform/resolvers/zod";
import { Link, createFileRoute, useNavigate } from "@tanstack/react-router";
import { CheckCircle2 } from "lucide-react";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { useSignUp } from "../api/generated/auth/auth";
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

export const Route = createFileRoute("/sign-up")({
  component: SignUpPage,
});

function buildSchema() {
  return z.object({
    email: z.email(m.auth_validation_email()),
    password: z
      .string()
      .min(8, m.auth_validation_password_length())
      .max(128, m.auth_validation_password_length()),
  });
}

type SignUpForm = z.infer<ReturnType<typeof buildSchema>>;

function SignUpPage() {
  const navigate = useNavigate();
  const [succeeded, setSucceeded] = useState(false);
  const schema = buildSchema();
  const form = useForm<SignUpForm>({
    resolver: zodResolver(schema),
    defaultValues: { email: "", password: "" },
  });

  const signUp = useSignUp({
    mutation: {
      onSuccess: () => setSucceeded(true),
      onError: (error) => applyAuthApiError(form, error),
    },
  });

  function onSubmit(values: SignUpForm) {
    signUp.mutate({ data: values });
  }

  if (succeeded) {
    return (
      <AuthLayout title={m.auth_sign_up_success_title()}>
        <Alert variant="success">
          <CheckCircle2 />
          <AlertDescription>{m.auth_sign_up_success_body()}</AlertDescription>
        </Alert>
        <Button
          className="mt-5 w-full"
          onClick={() => void navigate({ to: "/sign-in" })}
        >
          {m.auth_sign_up_success_action()}
        </Button>
      </AuthLayout>
    );
  }

  return (
    <AuthLayout
      title={m.auth_sign_up_title()}
      subtitle={m.auth_sign_up_subtitle()}
      footer={
        <span>
          {m.auth_sign_up_have_account()}{" "}
          <Link
            to="/sign-in"
            className="font-medium text-foreground hover:underline"
          >
            {m.auth_sign_up_sign_in()}
          </Link>
        </span>
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

          <FormField
            control={form.control}
            name="password"
            render={({ field }) => (
              <FormItem>
                <FormLabel>{m.auth_field_password()}</FormLabel>
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

          <Button type="submit" className="w-full" disabled={signUp.isPending}>
            {signUp.isPending
              ? m.auth_sign_up_submitting()
              : m.auth_sign_up_submit()}
          </Button>
        </form>
      </Form>
    </AuthLayout>
  );
}
