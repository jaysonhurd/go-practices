#!/usr/bin/env python3
"""Run the same repository and Go checks locally and in CI (Python 3.9+)."""

import subprocess
import sys
from pathlib import Path


def validate_go(root):
    """Check formatting and each nonignored Go module, failing on any error."""
    names = subprocess.check_output(
        ["git", "ls-files", "--cached", "--others", "--exclude-standard", "-z"], cwd=root
    ).decode().split("\0")
    files = sorted({name for name in names if name and (root / name).is_file()
                    and not {"vendor", ".git"}.intersection(Path(name).parts)})
    for name in files:
        if name.endswith(".go"):
            output = subprocess.check_output(["gofmt", "-l", name], cwd=root, text=True)
            if output.strip():
                raise ValueError(f"Unformatted Go file: {name}; run gofmt -w {name}")
    for name in files:
        if Path(name).name == "go.mod":
            module = (root / name).parent
            for arguments in (["build", "./..."], ["test", "-race", "-cover", "./..."], ["vet", "./..."]):
                subprocess.run(["go", *arguments], cwd=module, check=True)


def main():
    """Validate scripts, repository content, and all discovered Go modules."""
    root = Path(__file__).resolve().parents[1]
    subprocess.run([sys.executable, "-m", "unittest", "discover", "-s", "scripts", "-p", "test_*.py"], cwd=root, check=True)
    subprocess.run([sys.executable, "scripts/check_repository.py"], cwd=root, check=True)
    validate_go(root)


if __name__ == "__main__":
    try:
        main()
    except (subprocess.CalledProcessError, OSError, ValueError) as error:
        sys.exit(str(error))
