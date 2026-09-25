import "@typespec/http";
import "@typespec/openapi";
import "../../common/models.tsp";
import "../../common/responses.tsp";
import "./models.tsp";

using TypeSpec.OpenAPI;
using Starter.Common;

namespace Starter.Announcements;

@TypeSpec.Http.route("/api/announcements")
@tag("Announcements")
interface Announcements {
  @TypeSpec.Http.get
  @operationId("listAnnouncements")
  @summary("List announcements")
  list(
    ...PaginationQuery,
    @TypeSpec.Http.query status?: AnnouncementStatus,
  ): Page<Announcement> | BadRequestResponse;

  @TypeSpec.Http.post
  @operationId("createAnnouncement")
  @summary("Create an announcement")
  create(
    @TypeSpec.Http.body input: AnnouncementInput,
  ): CreatedResponse<Announcement> | BadRequestResponse;

  @TypeSpec.Http.route("/{id}")
  @TypeSpec.Http.get
  @operationId("getAnnouncement")
  @summary("Get an announcement")
  get(
    @TypeSpec.Http.path @format("uuid") id: string,
  ): Announcement | BadRequestResponse | NotFoundResponse;

  @TypeSpec.Http.route("/{id}")
  @TypeSpec.Http.put
  @operationId("updateAnnouncement")
  @summary("Update an announcement")
  update(
    @TypeSpec.Http.path @format("uuid") id: string,
    @TypeSpec.Http.body input: AnnouncementInput,
  ): Announcement | BadRequestResponse | NotFoundResponse;

  @TypeSpec.Http.route("/{id}")
  @TypeSpec.Http.delete
  @operationId("deleteAnnouncement")
  @summary("Delete an announcement")
  delete(
    @TypeSpec.Http.path @format("uuid") id: string,
  ): NoContentResponse | BadRequestResponse | NotFoundResponse;
}
