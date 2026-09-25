import "@typespec/http";
import "@typespec/openapi";
import "@typespec/openapi3";

import "./common/models.tsp";
import "./common/responses.tsp";
import "./features/announcements/routes.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;
using Starter.Common;

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

@route("/api/readyz")
@get
@operationId("getReadiness")
@summary("Check service readiness")
op getReadiness(): HealthResponse | ServiceUnavailableResponse;
