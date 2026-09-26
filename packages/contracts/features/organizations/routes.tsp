import "@typespec/http";
import "@typespec/openapi";
import "../../common/models.tsp";
import "../../common/responses.tsp";
import "./models.tsp";

using TypeSpec.OpenAPI;
using Starter.Common;

namespace Starter.Organizations;

@TypeSpec.Http.route("/api/organizations")
@tag("Organizations")
interface Organizations {
  @TypeSpec.Http.get
  @operationId("listUserOrganizations")
  @summary("List organizations for the current user")
  listUserOrganizations(): OrganizationWithRole[] | UnauthorizedResponse;

  @TypeSpec.Http.post
  @operationId("createOrganization")
  @summary("Create an organization")
  createOrganization(@TypeSpec.Http.body input: CreateOrganizationInput):
    | CreatedResponse<Organization>
    | BadRequestResponse
    | ConflictResponse
    | UnauthorizedResponse;

  @TypeSpec.Http.route("/invitations/accept")
  @TypeSpec.Http.post
  @operationId("acceptInvitation")
  @summary("Accept an organization invitation")
  acceptInvitation(@TypeSpec.Http.body input: AcceptInvitationInput):
    | OrganizationMembership
    | BadRequestResponse
    | NotFoundResponse
    | UnauthorizedResponse;

  @TypeSpec.Http.route("/{organizationId}")
  @TypeSpec.Http.get
  @operationId("getOrganization")
  @summary("Get organization details")
  getOrganization(@TypeSpec.Http.path @format("uuid") organizationId: string):
    | Organization
    | NotFoundResponse
    | UnauthorizedResponse
    | ForbiddenResponse;

  @TypeSpec.Http.route("/{organizationId}")
  @TypeSpec.Http.put
  @operationId("updateOrganization")
  @summary("Update organization details")
  updateOrganization(
    @TypeSpec.Http.path @format("uuid") organizationId: string,
    @TypeSpec.Http.body input: UpdateOrganizationInput,
  ):
    | Organization
    | BadRequestResponse
    | NotFoundResponse
    | ConflictResponse
    | UnauthorizedResponse
    | ForbiddenResponse;

  @TypeSpec.Http.route("/{organizationId}")
  @TypeSpec.Http.delete
  @operationId("deleteOrganization")
  @summary("Delete an organization")
  deleteOrganization(
    @TypeSpec.Http.path @format("uuid") organizationId: string,
  ):
    | NoContentResponse
    | NotFoundResponse
    | UnauthorizedResponse
    | ForbiddenResponse;

  @TypeSpec.Http.route("/{organizationId}/members")
  @TypeSpec.Http.get
  @operationId("listMembers")
  @summary("List organization members")
  listMembers(@TypeSpec.Http.path @format("uuid") organizationId: string):
    | OrganizationMembership[]
    | NotFoundResponse
    | UnauthorizedResponse
    | ForbiddenResponse;

  @TypeSpec.Http.route("/{organizationId}/members/{userId}")
  @TypeSpec.Http.put
  @operationId("updateMemberRole")
  @summary("Update a member's role")
  updateMemberRole(
    @TypeSpec.Http.path @format("uuid") organizationId: string,
    @TypeSpec.Http.path @format("uuid") userId: string,
    @TypeSpec.Http.body input: UpdateMembershipRoleInput,
  ):
    | OrganizationMembership
    | BadRequestResponse
    | NotFoundResponse
    | UnauthorizedResponse
    | ForbiddenResponse;

  @TypeSpec.Http.route("/{organizationId}/members/{userId}")
  @TypeSpec.Http.delete
  @operationId("removeMember")
  @summary("Remove a member from the organization")
  removeMember(
    @TypeSpec.Http.path @format("uuid") organizationId: string,
    @TypeSpec.Http.path @format("uuid") userId: string,
  ):
    | NoContentResponse
    | BadRequestResponse
    | NotFoundResponse
    | UnauthorizedResponse
    | ForbiddenResponse;

  @TypeSpec.Http.route("/{organizationId}/invitations")
  @TypeSpec.Http.get
  @operationId("listInvitations")
  @summary("List active organization invitations")
  listInvitations(@TypeSpec.Http.path @format("uuid") organizationId: string):
    | OrganizationInvitation[]
    | NotFoundResponse
    | UnauthorizedResponse
    | ForbiddenResponse;

  @TypeSpec.Http.route("/{organizationId}/invitations")
  @TypeSpec.Http.post
  @operationId("createInvitation")
  @summary("Invite a member to the organization")
  createInvitation(
    @TypeSpec.Http.path @format("uuid") organizationId: string,
    @TypeSpec.Http.body input: CreateInvitationInput,
  ):
    | CreatedResponse<CreatedInvitationResponse>
    | BadRequestResponse
    | ConflictResponse
    | NotFoundResponse
    | UnauthorizedResponse
    | ForbiddenResponse;

  @TypeSpec.Http.route("/{organizationId}/invitations/{invitationId}")
  @TypeSpec.Http.delete
  @operationId("revokeInvitation")
  @summary("Revoke an organization invitation")
  revokeInvitation(
    @TypeSpec.Http.path @format("uuid") organizationId: string,
    @TypeSpec.Http.path @format("uuid") invitationId: string,
  ):
    | NoContentResponse
    | NotFoundResponse
    | UnauthorizedResponse
    | ForbiddenResponse;
}
