import "@typespec/http";
import "@typespec/openapi";

using TypeSpec.Http;

namespace Starter.Common;

@friendlyName("Page_{name}", T)
model Page<T> {
  @doc("当前分页的项目列表")
  items: T[];

  @doc("符合过滤条件的总记录数")
  total: int64;

  @doc("分页大小限制")
  limit: int32;

  @doc("分页偏移量")
  offset: int32;
}

model PaginationQuery {
  @query
  @minValue(1)
  @maxValue(100)
  limit?: int32 = 20;

  @query
  @minValue(0)
  offset?: int32 = 0;
}
