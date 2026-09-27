import "@typespec/http";
import "@typespec/openapi";
import "./permissions.js";

using TypeSpec.OpenAPI;

namespace Starter.Common;

/**
 * Declares the Casbin `resource:action` permission an operation requires.
 * Emits the OpenAPI `x-permission` extension, consumed by
 * tools/generate-server.mjs to produce permissions_gen.go.
 */
extern dec permission(target: unknown, value: valueof string);
