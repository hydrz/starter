import { existsSync, readFileSync, readdirSync, statSync } from "node:fs";
import { dirname, extname, isAbsolute, join, relative, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const repositoryRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const documentationRoot = join(repositoryRoot, "docs");
const markdownFiles = [
  join(repositoryRoot, "README.md"),
  join(repositoryRoot, "CONTRIBUTING.md"),
  join(repositoryRoot, "AGENTS.md"),
  ...walk(documentationRoot).filter((file) => extname(file) === ".md"),
];
const failures = [];

for (const file of markdownFiles) {
  const content = readFileSync(file, "utf8");
  const displayPath = relative(repositoryRoot, file);
  const h1Count = content.match(/^# .+$/gm)?.length ?? 0;
  if (h1Count !== 1) {
    failures.push(`${displayPath}: expected exactly one level-one heading, found ${h1Count}`);
  }

  for (const match of content.matchAll(/!?\[[^\]]*\]\(([^)]+)\)/g)) {
    const rawTarget = match[1].trim().replace(/^<|>$/g, "");
    const target = rawTarget.split(/\s+/)[0];
    if (!target || target.startsWith("#") || /^[a-z][a-z\d+.-]*:/i.test(target)) {
      continue;
    }

    const pathPart = decodeURIComponent(target.split("#", 1)[0]);
    const absoluteTarget = isAbsolute(pathPart)
      ? join(repositoryRoot, pathPart.slice(1))
      : resolve(dirname(file), pathPart);
    if (!existsSync(absoluteTarget)) {
      failures.push(`${displayPath}: broken link ${rawTarget}`);
    }
  }
}

if (failures.length > 0) {
  console.error("Documentation checks failed:\n" + failures.map((failure) => `- ${failure}`).join("\n"));
  process.exit(1);
}

console.log(`Documentation checks passed for ${markdownFiles.length} files.`);

function walk(directory) {
  return readdirSync(directory)
    .flatMap((entry) => {
      const path = join(directory, entry);
      return statSync(path).isDirectory() ? walk(path) : [path];
    })
    .sort();
}
