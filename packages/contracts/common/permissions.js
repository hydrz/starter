/**
 * Implementation of the @permission decorator (common/permissions.tsp).
 * Delegates to TypeSpec.OpenAPI's @extension so the value is emitted as the
 * OpenAPI `x-permission` extension on the decorated operation.
 */
import { $extension } from "@typespec/openapi";

export const namespace = "Starter.Common";

export function $permission(context, target, value) {
  $extension(context, target, "x-permission", value);
}
