import { execFileSync } from "node:child_process";
import { readFileSync, rmSync, writeFileSync } from "node:fs";
import { extname, join } from "node:path";

const repository = valueFor("--repository") ?? process.env.GITHUB_REPOSITORY;
if (!repository || !/^[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+$/.test(repository)) {
  fail("pass --repository owner/name or set GITHUB_REPOSITORY");
}

const [owner, repositoryName] = repository.split("/");
const slug = normalizeSlug(repositoryName);
const scope = normalizeScope(repositoryName);
const displayName = valueFor("--display-name") ?? toDisplayName(repositoryName);
const modulePath = valueFor("--module") ?? `github.com/${owner}/${repositoryName}`;

if (repository === "hydrz/starter" && !process.argv.includes("--allow-source")) {
  fail("refusing to initialize the source template repository");
}

const replacements = [
  ["github.com/hydrz/starter", modulePath],
  ["@starter/", `@${scope}/`],
  ["Starter API", `${displayName} API`],
  ["Starter", displayName],
  ["starter", slug],
];
const excluded = new Set([
  ".github/template-bootstrap.json",
  ".github/workflows/template-bootstrap.yml",
  "tools/initialize-template.mjs",
]);
const changed = [];

for (const file of trackedFiles()) {
  if (excluded.has(file) || isLikelyBinary(file)) continue;
  let content;
  try {
    content = readFileSync(file, "utf8");
  } catch {
    continue;
  }
  const updated = replacements.reduce(
    (result, [source, target]) => result.replaceAll(source, target),
    content,
  );
  if (updated !== content) {
    writeFileSync(file, updated);
    changed.push(file);
  }
}

for (const file of excluded) rmSync(file, { force: true });

console.log(
  JSON.stringify(
    { repository, modulePath, packageScope: `@${scope}`, projectSlug: slug, displayName, changed },
    null,
    2,
  ),
);

function valueFor(flag) {
  const index = process.argv.indexOf(flag);
  if (index === -1) return undefined;
  const value = process.argv[index + 1];
  if (!value || value.startsWith("--")) fail(`${flag} requires a value`);
  return value;
}

function normalizeSlug(value) {
  const normalized = value.toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-|-$/g, "");
  if (!normalized) fail("repository name must contain a letter or digit");
  return normalized;
}

function normalizeScope(value) {
  return normalizeSlug(value).replace(/-+/g, "-");
}

function toDisplayName(value) {
  return value
    .split(/[-_.]+/)
    .filter(Boolean)
    .map((part) => part[0].toUpperCase() + part.slice(1))
    .join(" ");
}

function trackedFiles() {
  return execFileSync("git", ["ls-files", "-z"], { encoding: "utf8" }).split("\0").filter(Boolean);
}

function isLikelyBinary(file) {
  return [".png", ".jpg", ".jpeg", ".gif", ".ico", ".woff", ".woff2", ".pdf"].includes(
    extname(join(process.cwd(), file)).toLowerCase(),
  );
}

function fail(message) {
  console.error(`Template initialization failed: ${message}`);
  process.exit(1);
}
