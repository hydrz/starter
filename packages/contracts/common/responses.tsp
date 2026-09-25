import "@typespec/http";
import "@typespec/openapi";

namespace Starter.Common;

@friendlyName("ApiError")
model ApiError {
  code: string;
  message: string;
}

model CreatedResponse<T> {
  @TypeSpec.Http.statusCode statusCode: 201;
  @TypeSpec.Http.body body: T;
}

model NoContentResponse {
  @TypeSpec.Http.statusCode statusCode: 204;
}

model BadRequestResponse {
  @TypeSpec.Http.statusCode statusCode: 400;
  @TypeSpec.Http.body body: ApiError;
}

model NotFoundResponse {
  @TypeSpec.Http.statusCode statusCode: 404;
  @TypeSpec.Http.body body: ApiError;
}

model ServiceUnavailableResponse {
  @TypeSpec.Http.statusCode statusCode: 503;
  @TypeSpec.Http.body body: ApiError;
}
