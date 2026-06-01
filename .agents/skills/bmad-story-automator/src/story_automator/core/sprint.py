from __future__ import annotations

import re
from dataclasses import dataclass

from .story_keys import contextual_epic_story_keys, normalize_story_key, sprint_status_file
from .utils import file_exists, read_text, trim_lines


@dataclass(frozen=True)
class SprintStatus:
    found: bool
    story: str
    status: str
    done: bool
    reason: str = ""


def sprint_status_get(project_root: str, story_key: str, state_file: str | None = None) -> SprintStatus:
    status_file = sprint_status_file(project_root)
    if not file_exists(status_file):
        return SprintStatus(False, story_key, "unknown", False, "sprint-status.yaml not found")
    content = read_text(status_file)
    exact = _sprint_status_exact(content, story_key)
    if exact is not None:
        return exact
    norm = normalize_story_key(project_root, story_key, state_file=state_file)
    if norm is not None and norm.key != story_key:
        contextual = _sprint_status_exact(content, norm.key)
        if contextual is not None:
            return contextual
    return sprint_status_get_from_text(content, story_key)


def sprint_status_get_from_file(status_file: str, story_key: str) -> SprintStatus:
    if not file_exists(status_file):
        return SprintStatus(False, story_key, "unknown", False, "sprint-status.yaml not found")
    return sprint_status_get_from_text(read_text(status_file), story_key)


def sprint_status_get_from_text(content: str, story_key: str) -> SprintStatus:
    exact = _sprint_status_exact(content, story_key)
    if exact is not None:
        return exact
    prefix = story_key
    if "." in story_key:
        prefix = story_key.replace(".", "-")
    elif re.fullmatch(r"\d+-\d+-.+", story_key):
        prefix = "-".join(story_key.split("-", 2)[:2])
    if re.fullmatch(r"\d+-\d+", prefix):
        prefix_match = re.search(rf"(?m)^\s*({re.escape(prefix)}-[^:\s]+)\s*:\s*(\S+)", content)
        if prefix_match:
            status = prefix_match.group(2).strip()
            return SprintStatus(True, prefix_match.group(1), status, status == "done")
    return SprintStatus(False, story_key, "not_found", False)


def sprint_status_epic(project_root: str, epic: str, state_file: str | None = None) -> tuple[list[str], int]:
    status_file = sprint_status_file(project_root)
    if not file_exists(status_file):
        return ([], 0)
    content = read_text(status_file)
    contextual_keys = contextual_epic_story_keys(project_root, epic, state_file=state_file)
    if contextual_keys:
        stories: list[str] = []
        seen: set[str] = set()
        done_count = 0
        for key in contextual_keys:
            if key in seen:
                continue
            status = _sprint_status_exact(content, key)
            if status is None:
                continue
            stories.append(key)
            seen.add(key)
            if status.done:
                done_count += 1
        if stories:
            return (stories, done_count)
    stories: list[str] = []
    seen: set[str] = set()
    done_count = 0
    for line in trim_lines(content):
        line = line.strip()
        if not line or line.startswith("#"):
            continue
        if not (line.startswith(f"{epic}.") or line.startswith(f"{epic}-")):
            continue
        parts = line.split(":", 1)
        if len(parts) < 2:
            continue
        key = parts[0].strip()
        if key in seen:
            continue
        stories.append(key)
        seen.add(key)
        status = parts[1].strip().split()
        if status and status[0] == "done":
            done_count += 1
    return (stories, done_count)


def _sprint_status_exact(content: str, story_key: str) -> SprintStatus | None:
    match = re.search(rf"(?m)^\s*{re.escape(story_key)}:\s*(\S+)", content)
    if not match:
        return None
    status = match.group(1).strip()
    return SprintStatus(True, story_key, status, status == "done")
