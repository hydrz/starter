import { zodResolver } from "@hookform/resolvers/zod";
import { Link, createFileRoute, useNavigate } from "@tanstack/react-router";
import { useForm } from "react-hook-form";
import { z } from "zod";
import {
  useBeginOAuthSignIn,
  usePasswordSignIn,
} from "../api/generated/auth/auth";
import { OAuthProvider } from "../api/generated/model";
import { GithubIcon } from "../components/GithubIcon";
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
import { Separator } from "../components/ui/separator";
import { AuthLayout } from "../features/auth/AuthLayout";
import { applyAuthApiError } from "../features/auth/form-error";
import { setIssuedTokens } from "../lib/auth-session";
import * as m from "../paraglide/messages";

export const Route = createFileRoute("/sign-in")({
  component: SignInPage,
});

function buildSchema() {
  return z.object({
    email: z.email(m.auth_validation_email()),
    password: z.string().min(1, m.auth_validation_password_required()),
  });
}

type SignInForm = z.infer<ReturnType<typeof buildSchema>>;

function GoogleIcon({ size = 15 }: { size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" aria-hidden="true">
      <path
        fill="#4285F4"
        d="M23.49 12.27c0-.79-.07-1.54-.2-2.27H12v4.3h6.47a5.53 5.53 0 0 1-2.4 3.63v3h3.88c2.27-2.09 3.54-5.17 3.54-8.66Z"
      />
      <path
        fill="#34A853"
        d="M12 24c3.24 0 5.95-1.07 7.93-2.9l-3.88-3.01c-1.08.72-2.46 1.15-4.05 1.15-3.11 0-5.75-2.1-6.69-4.93H1.3v3.1A12 12 0 0 0 12 24Z"
      />
      <path
        fill="#FBBC05"
        d="M5.31 14.31A7.2 7.2 0 0 1 4.93 12c0-.8.14-1.58.38-2.31v-3.1H1.3A12 12 0 0 0 0 12c0 1.94.46 3.77 1.3 5.4l4.01-3.09Z"
      />
      <path
        fill="#EA4335"
        d="M12 4.76c1.76 0 3.34.6 4.59 1.79l3.44-3.44C17.94 1.19 15.23 0 12 0A12 12 0 0 0 1.3 6.6l4.01 3.1c.94-2.84 3.58-4.94 6.69-4.94Z"
      />
    </svg>
  );
}

function OAuthButtons() {
  const beginOAuth = useBeginOAuthSignIn();

  function startOAuth(
    provider: (typeof OAuthProvider)[keyof typeof OAuthProvider],
  ) {
    beginOAuth.mutate(
      { provider },
      {
        onSuccess: (res) => {
          if (res.status === 200) {
            window.location.href = res.data.authorizationUrl;
          }
        },
      },
    );
  }

  return (
    <div className="grid gap-2.5">
      <Button
        type="button"
        variant="outline"
        className="w-full"
        disabled={beginOAuth.isPending}
        onClick={() => startOAuth(OAuthProvider.google)}
      >
        <GoogleIcon />
        <span>{m.auth_sign_in_oauth_google()}</span>
      </Button>
      <Button
        type="button"
        variant="outline"
        className="w-full"
        disabled={beginOAuth.isPending}
        onClick={() => startOAuth(OAuthProvider.github)}
      >
        <GithubIcon size={15} />
        <span>{m.auth_sign_in_oauth_github()}</span>
      </Button>
    </div>
  );
}

function SignInPage() {
  const navigate = useNavigate();
  const schema = buildSchema();
  const form = useForm<SignInForm>({
    resolver: zodResolver(schema),
    defaultValues: { email: "", password: "" },
  });

  const passwordSignIn = usePasswordSignIn({
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

  function onSubmit(values: SignInForm) {
    passwordSignIn.mutate({ data: values });
  }

  return (
    <AuthLayout
      title={m.auth_sign_in_title()}
      subtitle={m.auth_sign_in_subtitle()}
      footer={
        <span>
          {m.auth_sign_in_no_account()}{" "}
          <Link
            to="/sign-up"
            className="font-medium text-foreground hover:underline"
          >
            {m.auth_sign_in_create_account()}
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
                <div className="flex items-center justify-between">
                  <FormLabel>{m.auth_field_password()}</FormLabel>
                  <Link
                    to="/forgot-password"
                    className="text-xs font-medium text-muted-foreground hover:text-foreground"
                  >
                    {m.auth_sign_in_forgot_password()}
                  </Link>
                </div>
                <FormControl>
                  <Input
                    type="password"
                    autoComplete="current-password"
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
            disabled={passwordSignIn.isPending}
          >
            {passwordSignIn.isPending
              ? m.auth_sign_in_submitting()
              : m.auth_sign_in_submit()}
          </Button>
        </form>
      </Form>

      <div className="my-5 flex items-center gap-3">
        <Separator className="flex-1" />
        <span className="text-xs text-muted-foreground">
          {m.auth_or_divider()}
        </span>
        <Separator className="flex-1" />
      </div>

      <div className="grid gap-2.5">
        <Link to="/otp/verify">
          <Button type="button" variant="outline" className="w-full">
            {m.auth_sign_in_with_otp()}
          </Button>
        </Link>
        <OAuthButtons />
      </div>
    </AuthLayout>
  );
}
