import "@typespec/http";
import "@typespec/openapi";
import "@typespec/openapi3";

import "./common/models.tsp";
import "./common/responses.tsp";
import "./features/system/routes.tsp";
import "./features/announcements/routes.tsp";
import "./features/auth/routes.tsp";
import "./features/organizations/routes.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;
using Starter.Common;

@service(#{ title: "Starter API" })
@info(#{ version: "0.1.0" })
@server("http://localhost:8080", "Local development")
namespace Starter;
