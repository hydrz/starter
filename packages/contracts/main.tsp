import "@typespec/http";
import "@typespec/openapi";
import "@typespec/openapi3";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

@service(#{ title: "Starter API" })
@info(#{ version: "0.1.0" })
@server("http://localhost:8080", "Local development")
namespace Starter;

@doc("Service health information.")
model HealthResponse {
  @doc("Current service status.")
  status: "ok";
}

@route("/api/healthz")
@get
@operationId("getHealth")
@summary("Check service health")
op getHealth(): HealthResponse;

enum AnnouncementStatus {
  draft,
  published,
}

model Announcement {
  @format("uuid")
  id: string;

  title: string;
  content: string;
  status: AnnouncementStatus;
  createdAt: utcDateTime;
  updatedAt: utcDateTime;
}

model AnnouncementInput {
  @minLength(1)
  @maxLength(120)
  title: string;

  @minLength(1)
  @maxLength(10000)
  content: string;

  status: AnnouncementStatus;
}

model AnnouncementPage {
  items: Announcement[];
  total: int64;
  limit: int32;
  offset: int32;
}

model ApiError {
  code: string;
  message: string;
}

model CreatedAnnouncementResponse {
  @statusCode statusCode: 201;
  @body body: Announcement;
}

model NoContentResponse {
  @statusCode statusCode: 204;
}

model BadRequestResponse {
  @statusCode statusCode: 400;
  @body body: ApiError;
}

model NotFoundResponse {
  @statusCode statusCode: 404;
  @body body: ApiError;
}

model ServiceUnavailableResponse {
  @statusCode statusCode: 503;
  @body body: ApiError;
}

@route("/api/readyz")
@get
@operationId("getReadiness")
@summary("Check service readiness")
op getReadiness(): HealthResponse | ServiceUnavailableResponse;

@route("/api/announcements")
interface Announcements {
  @get
  @operationId("listAnnouncements")
  @summary("List announcements")
  list(
    @query limit?: int32 = 20,
    @query offset?: int32 = 0,
    @query status?: AnnouncementStatus,
  ): AnnouncementPage | BadRequestResponse;

  @post
  @operationId("createAnnouncement")
  @summary("Create an announcement")
  create(
    @body input: AnnouncementInput,
  ): CreatedAnnouncementResponse | BadRequestResponse;

  @route("/{id}")
  @get
  @operationId("getAnnouncement")
  @summary("Get an announcement")
  get(
    @path @format("uuid") id: string,
  ): Announcement | BadRequestResponse | NotFoundResponse;

  @route("/{id}")
  @put
  @operationId("updateAnnouncement")
  @summary("Update an announcement")
  update(
    @path @format("uuid") id: string,
    @body input: AnnouncementInput,
  ): Announcement | BadRequestResponse | NotFoundResponse;

  @route("/{id}")
  @delete
  @operationId("deleteAnnouncement")
  @summary("Delete an announcement")
  delete(
    @path @format("uuid") id: string,
  ): NoContentResponse | BadRequestResponse | NotFoundResponse;
}
