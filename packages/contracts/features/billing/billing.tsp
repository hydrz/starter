import "@typespec/http";
import "@typespec/openapi";
import "@typespec/openapi3";

import "../../common/models.tsp";
import "../../common/responses.tsp";
import "./models.tsp";
import "./routes.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;
using Starter.Common;

@service(#{ title: "Billing API" })
@info(#{ version: "0.1.0" })
@server("http://localhost:8080", "Local development")
namespace Starter.Billing;
