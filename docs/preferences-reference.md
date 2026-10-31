> Preserved from the original repository README. The instructions below describe
> the separate coding-preferences repository, not setup commands for go-practices.

# coding-preferences

Personal Codex instructions and reusable project templates. Keep general working
preferences in the root `AGENTS.md` and project starting points under `templates/`.
Reusable skill skeletons live under `skills/`.

## Directory structure

```text
README.md                       Usage and maintenance instructions
AGENTS.md                       Personal defaults for Codex
skills/
  initialize-go/
    SKILL.md                    Go initialization skill skeleton (not implemented)
    assets/
      templates/                Reserved for generated project templates
templates/
  go-project/
    AGENTS.md                   Starting point for Go project instructions
```

## Install personal instructions

Run the following manually from this repository's root when ready to install:

```sh
mkdir -p ~/.codex
cp -i AGENTS.md ~/.codex/AGENTS.md
```

If the destination exists, review and merge any instructions you want to retain
before replacing it; `cp -i` prompts before overwriting. These examples assume the
default Codex home, `~/.codex`. If you use `CODEX_HOME`, use that directory instead.

## Update the installed copy

The installed file is a separate copy; pulling changes does not update it.
From this repository's root, pull and review changes, then copy again:

```sh
git pull --ff-only
diff -u ~/.codex/AGENTS.md AGENTS.md
cp -i AGENTS.md ~/.codex/AGENTS.md
```

`diff` exits with status 1 when files differ; this is expected. Preserve any local
customizations by merging them before overwriting. Start a new Codex session after
updating instructions so it loads the changes.

## Use the Go project template

From this repository's root, copy the template to an existing project's root:

```sh
cp -i templates/go-project/AGENTS.md /path/to/project/AGENTS.md
```

Replace `/path/to/project` with the actual destination. If the project already has
an `AGENTS.md`, merge the relevant sections instead of discarding its guidance.
Replace all `[TODO: ...]` placeholders, supply the actual local development
commands, and remove sections that do not apply. The template does not create a Go
module, source files, or development services. Copies are independent; review and
merge future template changes manually.

## How instructions interact

Codex loads global instructions from its home directory, then project instructions
from the project root down to the current working directory. More specific
instructions take precedence when guidance conflicts. In each directory,
`AGENTS.override.md` takes precedence over `AGENTS.md`; a global override can also
take precedence over the installed personal defaults. See the
[official OpenAI documentation](https://learn.chatgpt.com/docs/agent-configuration/agents-md).

Use global instructions for personal defaults and a project's root `AGENTS.md`
for its architecture, commands, and conventions. The root instructions also apply
when working in this repository. The template is intended for copying into another
project; its presence here does not apply Go-specific guidance to unrelated projects.

These Markdown files guide Codex. They do not configure GoLand, change Go compiler
behavior, or enforce formatting or linter rules. Configure the IDE, tools, and CI
separately when enforcement is needed.
