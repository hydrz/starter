import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { AlertTriangle, Megaphone } from "lucide-react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import {
  getListAnnouncementsQueryKey,
  useCreateAnnouncement,
  useDeleteAnnouncement,
  useListAnnouncements,
  useUpdateAnnouncement,
} from "../../api/generated/announcements/announcements";
import { Alert, AlertDescription } from "../../components/ui/alert";
import { Badge } from "../../components/ui/badge";
import { Button } from "../../components/ui/button";
import { EmptyState } from "../../components/layout/EmptyState";
import { Input } from "../../components/ui/input";
import { Skeleton } from "../../components/ui/skeleton";
import { Textarea } from "../../components/ui/textarea";
import { toast } from "../../components/ui/toast";
import { getErrorMessage } from "../../lib/errors";
import { useOrgContext } from "../organizations/OrgContext";
import * as m from "../../paraglide/messages";

function buildAnnouncementSchema() {
  return z.object({
    title: z
      .string()
      .trim()
      .min(1, m.announcements_validation_title_required())
      .max(120, m.announcements_validation_title_max()),
    content: z
      .string()
      .trim()
      .min(1, m.announcements_validation_content_required())
      .max(10_000, m.announcements_validation_content_max()),
    status: z.enum(["draft", "published"]),
  });
}

type AnnouncementForm = z.infer<ReturnType<typeof buildAnnouncementSchema>>;

const initialValues: AnnouncementForm = {
  title: "",
  content: "",
  status: "draft",
};

/**
 * 公告管理（DESIGN.md §6.2）：从旧的扁平 `/app/announcements`
 * 迁移进组织范围（`organizationId` 现在来自 `/app/$orgSlug` 解析出的
 * 真实组织，不再是硬编码的占位 UUID），文案改走 Paraglide，
 * 补齐加载骨架/空状态/错误态三态。
 */
export function Announcements() {
  const { organizationId } = useOrgContext();
  const queryClient = useQueryClient();
  const announcements = useListAnnouncements(organizationId, {
    limit: 10,
    offset: 0,
  });
  const form = useForm<AnnouncementForm>({
    resolver: zodResolver(buildAnnouncementSchema()),
    defaultValues: initialValues,
  });

  const refresh = () =>
    queryClient.invalidateQueries({
      queryKey: getListAnnouncementsQueryKey(organizationId),
    });

  const createAnnouncement = useCreateAnnouncement({
    mutation: {
      onSuccess: async () => {
        form.reset(initialValues);
        await refresh();
        toast.success(m.announcements_create_success());
      },
      onError: (error) => {
        const message = getErrorMessage(error);
        form.setError("root", { type: "server", message });
        toast.error(message);
      },
    },
  });

  const updateAnnouncement = useUpdateAnnouncement({
    mutation: {
      onSuccess: async () => {
        await refresh();
        toast.success(m.announcements_update_success());
      },
      onError: (error) => toast.error(getErrorMessage(error)),
    },
  });

  const deleteAnnouncement = useDeleteAnnouncement({
    mutation: {
      onSuccess: async () => {
        await refresh();
        toast.success(m.announcements_delete_success());
      },
      onError: (error) => toast.error(getErrorMessage(error)),
    },
  });

  const page =
    announcements.data?.status === 200 ? announcements.data.data : undefined;
  const submit = form.handleSubmit((data) =>
    createAnnouncement.mutate({ organizationId, data }),
  );

  return (
    <section className="announcements" aria-labelledby="announcements-title">
      <div className="section-heading">
        <div>
          <span className="eyebrow">{m.announcements_eyebrow()}</span>
          <h3 id="announcements-title">{m.announcements_heading()}</h3>
        </div>
        <span className="section-meta">
          {page
            ? m.announcements_count_total({ total: page.total })
            : m.announcements_count_syncing()}
        </span>
      </div>

      <div className="announcement-grid">
        <form className="announcement-form" onSubmit={submit}>
          <label>
            <span>{m.announcements_form_title_label()}</span>
            <Input
              placeholder={m.announcements_form_title_placeholder()}
              {...form.register("title")}
            />
            {form.formState.errors.title && (
              <small>{form.formState.errors.title.message}</small>
            )}
          </label>
          <label>
            <span>{m.announcements_form_content_label()}</span>
            <Textarea
              rows={4}
              placeholder={m.announcements_form_content_placeholder()}
              {...form.register("content")}
            />
            {form.formState.errors.content && (
              <small>{form.formState.errors.content.message}</small>
            )}
          </label>
          <div className="form-actions">
            <select
              aria-label={m.announcements_form_status_aria()}
              {...form.register("status")}
            >
              <option value="draft">
                {m.announcements_form_status_draft()}
              </option>
              <option value="published">
                {m.announcements_form_status_published()}
              </option>
            </select>
            <Button type="submit" disabled={createAnnouncement.isPending}>
              {createAnnouncement.isPending
                ? m.announcements_form_submitting()
                : m.announcements_form_submit()}
            </Button>
          </div>
          {form.formState.errors.root && (
            <p className="form-error">{form.formState.errors.root.message}</p>
          )}
        </form>

        <div className="announcement-list" aria-live="polite">
          {announcements.isPending && (
            <div className="space-y-2">
              <Skeleton className="h-24 w-full" />
              <Skeleton className="h-24 w-full" />
            </div>
          )}

          {announcements.isError && (
            <Alert variant="destructive">
              <AlertTriangle />
              <AlertDescription>
                {getErrorMessage(announcements.error)}
              </AlertDescription>
            </Alert>
          )}

          {page?.items.length === 0 && (
            <EmptyState icon={Megaphone} title={m.announcements_empty()} />
          )}

          {page?.items.map((item) => (
            <article className="announcement-item" key={item.id}>
              <div>
                <Badge
                  variant={
                    item.status === "published" ? "default" : "secondary"
                  }
                >
                  {item.status === "published"
                    ? m.announcements_status_published()
                    : m.announcements_status_draft()}
                </Badge>
                <time dateTime={item.updatedAt}>
                  {new Date(item.updatedAt).toLocaleDateString()}
                </time>
              </div>
              <h4>{item.title}</h4>
              <p>{item.content}</p>
              <div className="item-actions">
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  disabled={updateAnnouncement.isPending}
                  onClick={() =>
                    updateAnnouncement.mutate({
                      organizationId,
                      id: item.id,
                      data: {
                        title: item.title,
                        content: item.content,
                        status: item.status === "draft" ? "published" : "draft",
                      },
                    })
                  }
                >
                  {item.status === "draft"
                    ? m.announcements_action_publish()
                    : m.announcements_action_unpublish()}
                </Button>
                <Button
                  type="button"
                  variant="destructive"
                  size="sm"
                  disabled={deleteAnnouncement.isPending}
                  onClick={() =>
                    deleteAnnouncement.mutate({ organizationId, id: item.id })
                  }
                >
                  {m.announcements_action_delete()}
                </Button>
              </div>
            </article>
          ))}
        </div>
      </div>
    </section>
  );
}
