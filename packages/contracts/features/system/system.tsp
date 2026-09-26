import "@typespec/http";
import "@typespec/openapi";
import "@typespec/openapi3";

import "../../common/responses.tsp";
import "./routes.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;
using Starter.Common;

@service(#{ title: "System API" })
@info(#{ version: "0.1.0" })
@server("http://localhost:8080", "Local development")
namespace Starter.System;
