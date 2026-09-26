import { execSync } from "node:child_process";
import { existsSync, readdirSync } from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const repositoryRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const specDir = join(repositoryRoot, "spec", "generated");

if (!existsSync(specDir)) {
  console.error(`Spec directory not found: ${specDir}`);
  process.exit(1);
}

const files = readdirSync(specDir);
for (const file of files) {
  if (file.endsWith(".openapi.yaml") && file !== "openapi.yaml") {
    const featureName = file.replace(".openapi.yaml", "");
    const pkgName = `${featureName}api`;
    const targetDir = join(repositoryRoot, "internal", "api", pkgName);
    console.log(`Generating Go server with ogen: ${featureName} -> ${pkgName}...`);
    execSync(
      `go tool ogen --target ${targetDir} --package ${pkgName} --clean ${join(specDir, file)}`,
      {
        cwd: repositoryRoot,
        stdio: "inherit",
      },
    );
  }
}
