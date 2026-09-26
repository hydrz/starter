import "@typespec/openapi";

namespace Starter.Organizations;

@friendlyName("OrganizationRole")
enum OrganizationRole {
  owner,
  admin,
  member,
  viewer,
}

@friendlyName("Organization")
model Organization {
  @format("uuid")
  id: string;

  @minLength(1)
  @maxLength(100)
  slug: string;

  @minLength(1)
  @maxLength(120)
  name: string;

  createdAt: utcDateTime;
  updatedAt: utcDateTime;
}

@friendlyName("OrganizationWithRole")
model OrganizationWithRole {
  @format("uuid")
  id: string;

  @minLength(1)
  @maxLength(100)
  slug: string;

  @minLength(1)
  @maxLength(120)
  name: string;

  role: OrganizationRole;
  createdAt: utcDateTime;
  updatedAt: utcDateTime;
}

@friendlyName("CreateOrganizationInput")
model CreateOrganizationInput {
  @minLength(1)
  @maxLength(100)
  slug: string;

  @minLength(1)
  @maxLength(120)
  name: string;
}

@friendlyName("UpdateOrganizationInput")
model UpdateOrganizationInput {
  @minLength(1)
  @maxLength(100)
  slug: string;

  @minLength(1)
  @maxLength(120)
  name: string;
}

@friendlyName("OrganizationMembership")
model OrganizationMembership {
  @format("uuid")
  id: string;

  @format("uuid")
  organizationId: string;

  @format("uuid")
  userId: string;

  @format("email")
  email?: string;

  role: OrganizationRole;
  createdAt: utcDateTime;
  updatedAt: utcDateTime;
}

@friendlyName("UpdateMembershipRoleInput")
model UpdateMembershipRoleInput {
  role: OrganizationRole;
}

@friendlyName("OrganizationInvitation")
model OrganizationInvitation {
  @format("uuid")
  id: string;

  @format("uuid")
  organizationId: string;

  @format("email")
  email: string;

  role: OrganizationRole;
  expiresAt: utcDateTime;
  createdAt: utcDateTime;
}

@friendlyName("CreateInvitationInput")
model CreateInvitationInput {
  @format("email")
  email: string;

  role: OrganizationRole;
}

@friendlyName("CreatedInvitationResponse")
model CreatedInvitationResponse {
  invitation: OrganizationInvitation;
  token: string;
}

@friendlyName("AcceptInvitationInput")
model AcceptInvitationInput {
  token: string;
}
