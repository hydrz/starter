import { existsSync, readFileSync, readdirSync, statSync } from "node:fs";
import { basename, join, relative, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const repositoryRoot = resolve(fileURLToPath(new URL("..", import.meta.url)));
const skillsRoot = join(repositoryRoot, ".agents", "skills");
const failures = [];
const names = new Set();

for (const directory of directoryEntries(skillsRoot)) {
  const skillDirectory = join(skillsRoot, directory);
  const skillFile = join(skillDirectory, "SKILL.md");
  const metadataFile = join(skillDirectory, "agents", "openai.yaml");

  if (!existsSync(skillFile)) {
    failures.push(`${display(skillDirectory)}: missing SKILL.md`);
    continue;
  }

  const content = readFileSync(skillFile, "utf8");
  const frontmatter = content.match(/^---\n([\s\S]*?)\n---\n/);
  const name = frontmatter?.[1].match(/^name:\s*(.+)$/m)?.[1].trim();
  const description = frontmatter?.[1].match(/^description:\s*(.+)$/m)?.[1].trim();

  if (!frontmatter) failures.push(`${display(skillFile)}: invalid YAML frontmatter`);
  if (name !== directory) failures.push(`${display(skillFile)}: name must equal ${directory}`);
  if (!/^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(name ?? "")) {
    failures.push(`${display(skillFile)}: invalid skill name`);
  }
  if (!description || description.length < 40) {
    failures.push(`${display(skillFile)}: description must explain task and trigger`);
  }
  if (names.has(name)) failures.push(`${display(skillFile)}: duplicate skill name ${name}`);
  if (name) names.add(name);
  if (/\bTODO\b/.test(content)) failures.push(`${display(skillFile)}: unresolved TODO`);
  if ((content.match(/^# .+$/gm)?.length ?? 0) !== 1) {
    failures.push(`${display(skillFile)}: expected exactly one level-one heading`);
  }

  if (!existsSync(metadataFile)) {
    failures.push(`${display(skillDirectory)}: missing agents/openai.yaml`);
  } else {
    const metadata = readFileSync(metadataFile, "utf8");
    for (const key of ["display_name", "short_description", "default_prompt"]) {
      if (!new RegExp(`^  ${key}:`, "m").test(metadata)) {
        failures.push(`${display(metadataFile)}: missing interface.${key}`);
      }
    }
    if (name && !metadata.includes(`$${name}`)) {
      failures.push(`${display(metadataFile)}: default prompt must invoke $${name}`);
    }
  }
}

if (failures.length > 0) {
  console.error("Agent Skill checks failed:\n" + failures.map((failure) => `- ${failure}`).join("\n"));
  process.exit(1);
}

console.log(`Agent Skill checks passed for ${names.size} skills.`);

function directoryEntries(directory) {
  if (!existsSync(directory)) return [];
  return readdirSync(directory)
    .filter((entry) => statSync(join(directory, entry)).isDirectory())
    .sort();
}

function display(path) {
  return relative(repositoryRoot, path) || basename(path);
}
