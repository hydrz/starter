import { zodResolver } from "@hookform/resolvers/zod";
import { Link, useNavigate } from "@tanstack/react-router";
import { useQueryClient } from "@tanstack/react-query";
import { AlertTriangle, ArrowRight, Building2 } from "lucide-react";
import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import {
  getListUserOrganizationsQueryKey,
  useCreateOrganization,
  useListUserOrganizations,
} from "../../api/generated/organizations/organizations";
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
import { Skeleton } from "../../components/ui/skeleton";
import { AuthLayout } from "../auth/AuthLayout";
import { applyAuthApiError } from "../auth/form-error";
import { getErrorMessage } from "../../lib/errors";
import * as m from "../../paraglide/messages";
import { buildOrgSchema, slugify, type OrgFormValues } from "./org-schema";

/**
 * `/app`（无 `orgSlug`）：DESIGN.md §6.2 的"组织选择/自动跳转"。
 * - 恰好一个组织：自动跳转到 `/app/$orgSlug`（注册即自动创建个人组织，
 *   这是最常见的路径）；
 * - 多个组织：列出全部组织供选择，同时可以创建新组织
 *   （`OrgSwitcher` 的"创建组织"也落在这个页面）；
 * - 零个组织（正常情况下不会出现，防御性处理）：直接展示创建组织表单。
 */
export function OrgPickerPage() {
  const navigate = useNavigate();
  const organizationsQuery = useListUserOrganizations();
  const organizations =
    organizationsQuery.data?.status === 200
      ? organizationsQuery.data.data
      : undefined;

  const singleOrgSlug =
    organizations?.length === 1 ? organizations[0].slug : undefined;

  useEffect(() => {
    if (singleOrgSlug) {
      void navigate({
        to: "/app/$orgSlug",
        params: { orgSlug: singleOrgSlug },
        replace: true,
      });
    }
  }, [singleOrgSlug, navigate]);

  if (organizationsQuery.isPending || singleOrgSlug) {
    return (
      <AuthLayout title={m.org_picker_title()}>
        <div className="space-y-2.5">
          <Skeleton className="h-10 w-full" />
          <Skeleton className="h-10 w-full" />
        </div>
      </AuthLayout>
    );
  }

  if (organizationsQuery.isError) {
    return (
      <AuthLayout title={m.org_picker_title()}>
        <Alert variant="destructive">
          <AlertTriangle />
          <AlertDescription>
            {getErrorMessage(organizationsQuery.error)}
          </AlertDescription>
        </Alert>
        <Button
          className="mt-4 w-full"
          variant="outline"
          onClick={() => void organizationsQuery.refetch()}
        >
          {m.org_picker_retry()}
        </Button>
      </AuthLayout>
    );
  }

  const orgs = organizations ?? [];

  return (
    <AuthLayout
      title={
        orgs.length > 0 ? m.org_picker_title() : m.org_picker_empty_title()
      }
      subtitle={
        orgs.length > 0
          ? m.org_picker_subtitle_multi()
          : m.org_picker_empty_description()
      }
    >
      {orgs.length > 0 && (
        <div className="mb-5 grid gap-2">
          {orgs.map((org) => (
            <Link
              key={org.id}
              to="/app/$orgSlug"
              params={{ orgSlug: org.slug }}
              className="flex items-center justify-between rounded-lg border border-border bg-card px-3.5 py-2.5 text-sm font-medium text-foreground transition-colors hover:border-primary/60 hover:bg-muted/60"
            >
              <span className="flex items-center gap-2.5 truncate">
                <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-md bg-primary/10 text-primary">
                  <Building2 size={14} />
                </span>
                <span className="truncate">{org.name}</span>
              </span>
              <ArrowRight
                size={14}
                className="shrink-0 text-muted-foreground"
              />
            </Link>
          ))}
        </div>
      )}

      <CreateOrganizationForm
        variant={orgs.length > 0 ? "secondary" : "primary"}
      />
    </AuthLayout>
  );
}

function CreateOrganizationForm({
  variant,
}: {
  variant: "primary" | "secondary";
}) {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const schema = buildOrgSchema();
  const form = useForm<OrgFormValues>({
    resolver: zodResolver(schema),
    defaultValues: { name: "", slug: "" },
  });
  const [slugTouched, setSlugTouched] = useState(false);

  const createOrganization = useCreateOrganization({
    mutation: {
      onSuccess: async (res) => {
        if (res.status === 201) {
          await queryClient.invalidateQueries({
            queryKey: getListUserOrganizationsQueryKey(),
          });
          await navigate({
            to: "/app/$orgSlug",
            params: { orgSlug: res.data.slug },
          });
        }
      },
      onError: (error) => applyAuthApiError(form, error),
    },
  });

  const onSubmit = form.handleSubmit((values) =>
    createOrganization.mutate({ data: values }),
  );

  return (
    <Form {...form}>
      {variant === "secondary" && (
        <p className="mb-3 mt-1 text-center text-xs font-medium uppercase tracking-wide text-muted-foreground">
          {m.org_picker_create_secondary_heading()}
        </p>
      )}
      <form onSubmit={onSubmit} className="grid gap-3.5">
        <FormField
          control={form.control}
          name="name"
          render={({ field }) => (
            <FormItem>
              <FormLabel>{m.org_picker_create_name_label()}</FormLabel>
              <FormControl>
                <Input
                  {...field}
                  placeholder={m.org_picker_create_name_placeholder()}
                  onChange={(event) => {
                    field.onChange(event);
                    if (!slugTouched) {
                      form.setValue("slug", slugify(event.target.value), {
                        shouldValidate: form.formState.isSubmitted,
                      });
                    }
                  }}
                />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name="slug"
          render={({ field }) => (
            <FormItem>
              <FormLabel>{m.org_picker_create_slug_label()}</FormLabel>
              <FormControl>
                <Input
                  {...field}
                  placeholder={m.org_picker_create_slug_placeholder()}
                  onChange={(event) => {
                    setSlugTouched(true);
                    field.onChange(event);
                  }}
                />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        {form.formState.errors.root && (
          <Alert variant="destructive">
            <AlertTriangle />
            <AlertDescription>
              {form.formState.errors.root.message}
            </AlertDescription>
          </Alert>
        )}
        <Button
          type="submit"
          variant={variant === "primary" ? "default" : "outline"}
          disabled={createOrganization.isPending}
        >
          {createOrganization.isPending
            ? m.org_picker_create_submitting()
            : m.org_picker_create_submit()}
        </Button>
      </form>
    </Form>
  );
}
