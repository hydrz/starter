import { execSync } from "node:child_process";
import { copyFileSync, existsSync, mkdirSync, readdirSync } from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const contractsRoot = resolve(dirname(fileURLToPath(import.meta.url)));
const specGeneratedDir = resolve(contractsRoot, "../../spec/generated");
const serverAssetsDir = resolve(
  contractsRoot,
  "../../internal/platform/httpserver/assets",
);

mkdirSync(specGeneratedDir, { recursive: true });
mkdirSync(serverAssetsDir, { recursive: true });

function runTsp(args) {
  execSync(`npx tsp ${args}`, {
    cwd: contractsRoot,
    stdio: "inherit",
  });
}

console.log("Compiling unified OpenAPI specs...");
runTsp(
  `compile main.tsp --option "@typespec/openapi3.output-file=openapi.yaml"`,
);

// Sync openapi.yaml to internal/platform/httpserver/assets for embedded docs
copyFileSync(
  join(specGeneratedDir, "openapi.yaml"),
  join(serverAssetsDir, "openapi.yaml"),
);
console.log("  * Synced openapi.yaml to httpserver assets");

// Compile modular specs for each feature
const featuresDir = join(contractsRoot, "features");
if (existsSync(featuresDir)) {
  const entries = readdirSync(featuresDir, { withFileTypes: true });
  for (const entry of entries) {
    if (entry.isDirectory()) {
      const featureName = entry.name;
      const featureTsp = join(featuresDir, featureName, `${featureName}.tsp`);
      if (existsSync(featureTsp)) {
        console.log(`Compiling modular OpenAPI spec: ${featureName}...`);
        runTsp(
          `compile features/${featureName}/${featureName}.tsp --option "@typespec/openapi3.output-file=${featureName}.openapi.yaml"`,
        );
      }
    }
  }
}

console.log("Contracts build completed.");
