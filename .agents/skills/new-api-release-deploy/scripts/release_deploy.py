#!/usr/bin/env python3
"""Create a timestamped new-api release and watch the docker-build workflow."""

from __future__ import annotations

import argparse
import json
import re
import subprocess
import sys
import time
from dataclasses import dataclass
from datetime import datetime
from pathlib import Path
from typing import Iterable


REPO = "HuaqingAI/new-api"
WORKFLOW = "docker-build.yml"
BASE_VERSION_RE = re.compile(r"^v\d+\.\d+\.\d+$")


@dataclass(frozen=True)
class ReleaseNames:
    base_version: str
    timestamp: str
    upstream_version: str

    @property
    def tag(self) -> str:
        return f"{self.base_version}-{self.timestamp}"

    @property
    def app_version(self) -> str:
        return f"{self.tag}+new-api.{self.upstream_version}"


def run(
    args: Iterable[str],
    *,
    cwd: Path,
    check: bool = True,
    capture: bool = True,
) -> subprocess.CompletedProcess[str]:
    result = subprocess.run(
        list(args),
        cwd=cwd,
        text=True,
        capture_output=capture,
        check=False,
    )
    if check and result.returncode != 0:
        command = " ".join(args)
        stderr = (result.stderr or result.stdout or "").strip()
        raise SystemExit(f"Command failed ({result.returncode}): {command}\n{stderr}")
    return result


def find_repo_root(start: Path) -> Path:
    result = run(["git", "rev-parse", "--show-toplevel"], cwd=start)
    return Path(result.stdout.strip())


def normalize_base_version(value: str) -> str:
    value = value.strip()
    match = re.match(r"^(v\d+\.\d+\.\d+(?:[-+][0-9A-Za-z.-]+)?)(?:-\d{14}\+new-api\..+)?$", value)
    if match:
        value = match.group(1)
    if not BASE_VERSION_RE.match(value):
        raise SystemExit(
            "Base version must look like v0.1.1. "
            "Full release examples like v0.1.1-20260618125127+new-api.v1.0.0-rc.11 are also accepted."
        )
    return value


def read_upstream_version(repo_root: Path) -> str:
    version_file = repo_root / "UPSTREAM_VERSION"
    if not version_file.exists():
        raise SystemExit("UPSTREAM_VERSION is missing at the repository root")
    upstream = version_file.read_text(encoding="utf-8").strip()
    if not upstream:
        raise SystemExit("UPSTREAM_VERSION is empty")
    return upstream


def check_git_state(repo_root: Path, allow_dirty: bool, allow_non_dev: bool) -> None:
    branch = run(["git", "branch", "--show-current"], cwd=repo_root).stdout.strip()
    if branch != "dev" and not allow_non_dev:
        raise SystemExit(f"Current branch is {branch!r}; switch to dev before releasing")

    status = run(["git", "status", "--short"], cwd=repo_root).stdout.strip()
    if status and not allow_dirty:
        raise SystemExit("Worktree is not clean. Commit/push changes before releasing or pass --allow-dirty for testing.")

    run(["git", "fetch", "origin", "dev", "--tags"], cwd=repo_root)

    local = run(["git", "rev-parse", "HEAD"], cwd=repo_root).stdout.strip()
    remote = run(["git", "rev-parse", "origin/dev"], cwd=repo_root).stdout.strip()
    if branch == "dev" and local != remote:
        raise SystemExit("Local dev does not match origin/dev. Push or pull before releasing.")


def ensure_gh_auth(repo_root: Path) -> None:
    run(["gh", "auth", "status"], cwd=repo_root)


def create_release(repo_root: Path, names: ReleaseNames) -> str:
    if run(["git", "rev-parse", "-q", "--verify", f"refs/tags/{names.tag}"], cwd=repo_root, check=False).returncode == 0:
        raise SystemExit(f"Tag already exists locally: {names.tag}")

    remote_tag = run(
        ["git", "ls-remote", "--tags", "origin", f"refs/tags/{names.tag}"],
        cwd=repo_root,
    ).stdout.strip()
    if remote_tag:
        raise SystemExit(f"Tag already exists on origin: {names.tag}")

    body = (
        f"Automated release for {names.app_version}.\n\n"
        f"- Base version: {names.base_version}\n"
        f"- Timestamp: {names.timestamp}\n"
        f"- Upstream new-api version: {names.upstream_version}\n"
        f"- Source branch: dev\n"
    )
    run(
        [
            "gh",
            "release",
            "create",
            names.tag,
            "--repo",
            REPO,
            "--target",
            "dev",
            "--title",
            names.app_version,
            "--notes",
            body,
        ],
        cwd=repo_root,
        capture=False,
    )
    return f"https://github.com/{REPO}/releases/tag/{names.tag}"


def find_run(repo_root: Path, release_title: str, created_after_epoch: float) -> dict:
    deadline = time.time() + 180
    while time.time() < deadline:
        result = run(
            [
                "gh",
                "run",
                "list",
                "--repo",
                REPO,
                "--workflow",
                WORKFLOW,
                "--event",
                "release",
                "--limit",
                "20",
                "--json",
                "databaseId,displayTitle,status,conclusion,url,createdAt",
            ],
            cwd=repo_root,
        )
        runs = json.loads(result.stdout)
        for item in runs:
            created = item.get("createdAt", "")
            try:
                created_epoch = datetime.fromisoformat(created.replace("Z", "+00:00")).timestamp()
            except ValueError:
                created_epoch = 0
            if item.get("displayTitle") == release_title and created_epoch >= created_after_epoch - 30:
                return item
        time.sleep(5)
    raise SystemExit(f"Could not find a {WORKFLOW} release run for {release_title}")


def watch_run(repo_root: Path, run_id: int) -> dict:
    run(["gh", "run", "watch", str(run_id), "--repo", REPO, "--exit-status"], cwd=repo_root, check=False, capture=False)
    result = run(
        [
            "gh",
            "run",
            "view",
            str(run_id),
            "--repo",
            REPO,
            "--json",
            "status,conclusion,url,jobs",
        ],
        cwd=repo_root,
    )
    return json.loads(result.stdout)


def print_failed_logs_hint(repo_root: Path, run_id: int) -> None:
    failed = run(
        ["gh", "run", "view", str(run_id), "--repo", REPO, "--log-failed"],
        cwd=repo_root,
        check=False,
    )
    output = (failed.stdout or failed.stderr or "").strip()
    if output:
        print("\nFailed log excerpt:")
        print(output[-6000:])


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--base-version", required=True, help="Base version such as v0.1.1, or a full previous release name")
    parser.add_argument("--timestamp", help="Override timestamp for testing. Defaults to current local time in YYYYMMDDHHMMSS.")
    parser.add_argument("--dry-run", action="store_true", help="Print generated names without creating a release")
    parser.add_argument("--no-watch", action="store_true", help="Create the release but do not wait for the workflow")
    parser.add_argument("--allow-dirty", action="store_true", help="Allow a dirty worktree, intended only for local testing")
    parser.add_argument("--allow-non-dev", action="store_true", help="Allow running outside dev, intended only for local testing")
    args = parser.parse_args()

    repo_root = find_repo_root(Path.cwd())
    upstream = read_upstream_version(repo_root)
    timestamp = args.timestamp or datetime.now().strftime("%Y%m%d%H%M%S")
    if not re.fullmatch(r"\d{14}", timestamp):
        raise SystemExit("Timestamp must be YYYYMMDDHHMMSS")

    names = ReleaseNames(
        base_version=normalize_base_version(args.base_version),
        timestamp=timestamp,
        upstream_version=upstream,
    )

    print(f"Repository: {repo_root}")
    print(f"Base version: {names.base_version}")
    print(f"Upstream version: {names.upstream_version}")
    print(f"Release tag: {names.tag}")
    print(f"Release title: {names.app_version}")

    check_git_state(repo_root, args.allow_dirty, args.allow_non_dev)
    ensure_gh_auth(repo_root)

    if args.dry_run:
        print("Dry run only; no release created.")
        return 0

    created_after = time.time()
    release_url = create_release(repo_root, names)
    print(f"Release URL: {release_url}")

    run_info = find_run(repo_root, names.app_version, created_after)
    run_id = int(run_info["databaseId"])
    print(f"Workflow URL: {run_info['url']}")

    if args.no_watch:
        return 0

    final = watch_run(repo_root, run_id)
    print(f"Workflow conclusion: {final.get('conclusion')}")
    if final.get("conclusion") != "success":
        print_failed_logs_hint(repo_root, run_id)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
