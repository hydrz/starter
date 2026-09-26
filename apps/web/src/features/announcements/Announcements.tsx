import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { HttpError } from "../../api/client";
import {
  getListAnnouncementsQueryKey,
  useCreateAnnouncement,
  useDeleteAnnouncement,
  useListAnnouncements,
  useUpdateAnnouncement,
} from "../../api/generated/announcements/announcements";
import { Badge } from "../../components/ui/badge";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { Textarea } from "../../components/ui/textarea";

const announcementSchema = z.object({
  title: z
    .string()
    .trim()
    .min(1, "请输入公告标题")
    .max(120, "标题不能超过 120 个字符"),
  content: z
    .string()
    .trim()
    .min(1, "请输入公告内容")
    .max(10_000, "内容不能超过 10000 个字符"),
  status: z.enum(["draft", "published"]),
});

type AnnouncementForm = z.infer<typeof announcementSchema>;

const initialValues: AnnouncementForm = {
  title: "",
  content: "",
  status: "draft",
};

export function Announcements() {
  const queryClient = useQueryClient();
  const announcements = useListAnnouncements({ limit: 10, offset: 0 });
  const form = useForm<AnnouncementForm>({
    resolver: zodResolver(announcementSchema),
    defaultValues: initialValues,
  });

  const refresh = () =>
    queryClient.invalidateQueries({ queryKey: getListAnnouncementsQueryKey() });

  const createAnnouncement = useCreateAnnouncement({
    mutation: {
      onSuccess: async () => {
        form.reset(initialValues);
        await refresh();
      },
    },
  });

  const updateAnnouncement = useUpdateAnnouncement({
    mutation: {
      onSuccess: async () => {
        await refresh();
      },
    },
  });

  const deleteAnnouncement = useDeleteAnnouncement({
    mutation: {
      onSuccess: async () => {
        await refresh();
      },
    },
  });

  const page =
    announcements.data?.status === 200 ? announcements.data.data : undefined;
  const submit = form.handleSubmit((data) =>
    createAnnouncement.mutate({ data }),
  );

  const errorMessage = createAnnouncement.error
    ? createAnnouncement.error instanceof HttpError
      ? createAnnouncement.error.data?.message ||
        createAnnouncement.error.message
      : ((createAnnouncement.error as unknown as { message?: string })
          ?.message ?? "创建公告失败，请重试")
    : null;

  return (
    <section className="announcements" aria-labelledby="announcements-title">
      <div className="section-heading">
        <div>
          <span className="eyebrow">VERTICAL SLICE</span>
          <h3 id="announcements-title">公告通知</h3>
        </div>
        <span className="section-meta">
          {page ? `${page.total} TOTAL` : "SYNCING"}
        </span>
      </div>

      <div className="announcement-grid">
        <form className="announcement-form" onSubmit={submit}>
          <label>
            <span>公告标题</span>
            <Input
              placeholder="例如：计划维护通知"
              {...form.register("title")}
            />
            {form.formState.errors.title && (
              <small>{form.formState.errors.title.message}</small>
            )}
          </label>
          <label>
            <span>公告内容</span>
            <Textarea
              rows={4}
              placeholder="填写需要发布的内容"
              {...form.register("content")}
            />
            {form.formState.errors.content && (
              <small>{form.formState.errors.content.message}</small>
            )}
          </label>
          <div className="form-actions">
            <select aria-label="发布状态" {...form.register("status")}>
              <option value="draft">保存为草稿</option>
              <option value="published">立即发布</option>
            </select>
            <Button type="submit" disabled={createAnnouncement.isPending}>
              {createAnnouncement.isPending ? "正在保存…" : "创建公告"}
            </Button>
          </div>
          {errorMessage && <p className="form-error">{errorMessage}</p>}
        </form>

        <div className="announcement-list" aria-live="polite">
          {announcements.isPending && (
            <p className="empty-state">正在加载公告…</p>
          )}
          {announcements.isError && (
            <p className="empty-state error">公告加载失败，请稍后重试。</p>
          )}
          {page?.items.length === 0 && (
            <p className="empty-state">还没有公告，请创建第一条。</p>
          )}
          {page?.items.map((item) => (
            <article className="announcement-item" key={item.id}>
              <div>
                <Badge
                  variant={
                    item.status === "published" ? "default" : "secondary"
                  }
                >
                  {item.status === "published" ? "已发布" : "草稿"}
                </Badge>
                <time dateTime={item.updatedAt}>
                  {new Date(item.updatedAt).toLocaleDateString("zh-CN")}
                </time>
              </div>
              <h4>{item.title}</h4>
              <p>{item.content}</p>
              <div className="item-actions">
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() =>
                    updateAnnouncement.mutate({
                      id: item.id,
                      data: {
                        title: item.title,
                        content: item.content,
                        status: item.status === "draft" ? "published" : "draft",
                      },
                    })
                  }
                >
                  {item.status === "draft" ? "发布" : "转为草稿"}
                </Button>
                <Button
                  type="button"
                  variant="destructive"
                  size="sm"
                  onClick={() => deleteAnnouncement.mutate({ id: item.id })}
                >
                  删除
                </Button>
              </div>
            </article>
          ))}
        </div>
      </div>
    </section>
  );
}
