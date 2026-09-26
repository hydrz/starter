import "@typespec/http";
import "@typespec/openapi";
import "../../common/responses.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;
using Starter.Common;

namespace Starter.System;

@doc("Service health information.")
model HealthResponse {
  @doc("Current service status.")
  status: "ok";
}

@tag("System")
interface System {
  @route("/api/healthz")
  @get
  @operationId("getHealth")
  @summary("Check service health")
  getHealth(): HealthResponse;

  @route("/api/readyz")
  @get
  @operationId("getReadiness")
  @summary("Check service readiness")
  getReadiness(): HealthResponse | ServiceUnavailableResponse;
}
