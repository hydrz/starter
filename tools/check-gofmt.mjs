import { execFileSync } from "node:child_process";

const targets = process.argv.slice(2);
const checkDirs = targets.length > 0 ? targets : ["apps", "db", "internal"];

try {
  const output = execFileSync("gofmt", ["-l", ...checkDirs], {
    encoding: "utf8",
    stdio: ["ignore", "pipe", "pipe"],
  }).trim();

  if (output.length > 0) {
    const files = output.split(/\r?\n/).filter(Boolean);
    console.error("The following Go files are not formatted (run 'pnpm format:go'):");
    for (const file of files) {
      console.error(`  - ${file}`);
    }
    process.exit(1);
  }

  console.log(`Go format check passed for: ${checkDirs.join(", ")}`);
} catch (error) {
  if (error.code === "ENOENT") {
    console.error("gofmt is not found in PATH.");
    process.exit(1);
  }
  console.error("Failed to run gofmt:", error.message);
  process.exit(1);
}
