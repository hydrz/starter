import { existsSync, mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const repositoryRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");

const rawName = process.argv[2];
if (!rawName || !/^[a-z][a-z0-9_-]*$/i.test(rawName)) {
  console.error("Usage: node tools/scaffold-slice.mjs <feature-name>");
  console.error("Example: node tools/scaffold-slice.mjs orders");
  process.exit(1);
}

const featureName = rawName.toLowerCase().replace(/_/g, "-");
const singularName = featureName.endsWith("s")
  ? featureName.slice(0, -1)
  : featureName;
const pascalSingular = toPascalCase(singularName);
const pascalFeature = toPascalCase(featureName);
const timestamp = new Date()
  .toISOString()
  .replace(/\D/g, "")
  .slice(0, 14);

console.log(`Scaffolding vertical slice: ${featureName} (${pascalSingular})...`);

// 1. Contract Models
const contractDir = join(
  repositoryRoot,
  "packages",
  "contracts",
  "features",
  featureName,
);
mkdirSync(contractDir, { recursive: true });

const modelsTspPath = join(contractDir, "models.tsp");
if (!existsSync(modelsTspPath)) {
  writeFileSync(
    modelsTspPath,
    `import "@typespec/openapi";

namespace Starter.${pascalFeature};

@friendlyName("${pascalSingular}Status")
enum ${pascalSingular}Status {
  active,
  archived,
}

@friendlyName("${pascalSingular}")
model ${pascalSingular} {
  @format("uuid")
  id: string;

  title: string;
  status: ${pascalSingular}Status;
  createdAt: utcDateTime;
  updatedAt: utcDateTime;
}

@friendlyName("${pascalSingular}Input")
model ${pascalSingular}Input {
  @minLength(1)
  @maxLength(120)
  title: string;

  status: ${pascalSingular}Status;
}
`,
  );
  console.log(`  + ${modelsTspPath}`);
}

// 2. Contract Routes
const routesTspPath = join(contractDir, "routes.tsp");
if (!existsSync(routesTspPath)) {
  writeFileSync(
    routesTspPath,
    `import "@typespec/http";
import "@typespec/openapi";
import "../../common/models.tsp";
import "../../common/responses.tsp";
import "./models.tsp";

using TypeSpec.OpenAPI;
using Starter.Common;

namespace Starter.${pascalFeature};

@TypeSpec.Http.route("/api/${featureName}")
@tag("${pascalFeature}")
interface ${pascalFeature} {
  @TypeSpec.Http.get
  @operationId("list${pascalFeature}")
  @summary("List ${featureName}")
  list(
    ...PaginationQuery,
    @TypeSpec.Http.query status?: ${pascalSingular}Status,
  ): Page<${pascalSingular}> | BadRequestResponse;

  @TypeSpec.Http.post
  @operationId("create${pascalSingular}")
  @summary("Create a ${singularName}")
  create(
    @TypeSpec.Http.body input: ${pascalSingular}Input,
  ): CreatedResponse<${pascalSingular}> | BadRequestResponse;

  @TypeSpec.Http.route("/{id}")
  @TypeSpec.Http.get
  @operationId("get${pascalSingular}")
  @summary("Get a ${singularName}")
  get(
    @TypeSpec.Http.path @format("uuid") id: string,
  ): ${pascalSingular} | BadRequestResponse | NotFoundResponse;

  @TypeSpec.Http.route("/{id}")
  @TypeSpec.Http.delete
  @operationId("delete${pascalSingular}")
  @summary("Delete a ${singularName}")
  delete(
    @TypeSpec.Http.path @format("uuid") id: string,
  ): NoContentResponse | BadRequestResponse | NotFoundResponse;
}
`,
  );
  console.log(`  + ${routesTspPath}`);
}

// 3. Register in packages/contracts/main.tsp
const mainTspPath = join(repositoryRoot, "packages", "contracts", "main.tsp");
const mainTsp = readFileSync(mainTspPath, "utf8");
const importStmt = `import "./features/${featureName}/routes.tsp";`;
if (!mainTsp.includes(importStmt)) {
  const updatedMainTsp = mainTsp.replace(
    /(import "\.\/features\/[^"]+";\n)/,
    `$1${importStmt}\n`,
  );
  writeFileSync(mainTspPath, updatedMainTsp);
  console.log(`  * Updated ${mainTspPath}`);
}

// 4. Database Migration
const migrationPath = join(
  repositoryRoot,
  "db",
  "migrations",
  `${timestamp}_create_${featureName}.sql`,
);
writeFileSync(
  migrationPath,
  `-- +goose Up
CREATE TYPE ${singularName}_status AS ENUM ('active', 'archived');

CREATE TABLE ${featureName} (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    title text NOT NULL CHECK (char_length(title) BETWEEN 1 AND 120),
    status ${singularName}_status NOT NULL DEFAULT 'active',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX ${featureName}_created_at_idx ON ${featureName} (created_at DESC, id DESC);

-- +goose Down
DROP TABLE ${featureName};
DROP TYPE ${singularName}_status;
`,
);
console.log(`  + ${migrationPath}`);

// 5. Database Queries
const queryPath = join(repositoryRoot, "db", "queries", `${featureName}.sql`);
writeFileSync(
  queryPath,
  `-- name: Count${pascalFeature} :one
SELECT count(*)
FROM ${featureName}
WHERE sqlc.narg('status')::${singularName}_status IS NULL
   OR status = sqlc.narg('status')::${singularName}_status;

-- name: List${pascalFeature} :many
SELECT id, title, status, created_at, updated_at
FROM ${featureName}
WHERE sqlc.narg('status')::${singularName}_status IS NULL
   OR status = sqlc.narg('status')::${singularName}_status
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: Get${pascalSingular} :one
SELECT id, title, status, created_at, updated_at
FROM ${featureName}
WHERE id = $1;

-- name: Create${pascalSingular} :one
INSERT INTO ${featureName} (title, status)
VALUES ($1, $2)
RETURNING id, title, status, created_at, updated_at;

-- name: Delete${pascalSingular} :execrows
DELETE FROM ${featureName}
WHERE id = $1;
`,
);
console.log(`  + ${queryPath}`);

console.log(`
Done! Next steps:
  1. Review contracts and SQL queries
  2. Run 'pnpm generate' to generate server and client code
  3. Implement 'internal/${singularName}/' domain service and repository
`);

function toPascalCase(str) {
  return str
    .split(/[-_]+/)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join("");
}
