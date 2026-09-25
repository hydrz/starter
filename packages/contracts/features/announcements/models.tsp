import "@typespec/openapi";

namespace Starter.Announcements;

@friendlyName("AnnouncementStatus")
enum AnnouncementStatus {
  draft,
  published,
}

@friendlyName("Announcement")
model Announcement {
  @format("uuid")
  id: string;

  title: string;
  content: string;
  status: AnnouncementStatus;
  createdAt: utcDateTime;
  updatedAt: utcDateTime;
}

@friendlyName("AnnouncementInput")
model AnnouncementInput {
  @minLength(1)
  @maxLength(120)
  title: string;

  @minLength(1)
  @maxLength(10000)
  content: string;

  status: AnnouncementStatus;
}
