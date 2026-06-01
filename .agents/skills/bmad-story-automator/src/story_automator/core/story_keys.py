from __future__ import annotations

import json
import os
import re
from dataclasses import dataclass
from pathlib import Path

from .frontmatter import parse_simple_frontmatter
from .runtime_layout import active_marker_path
from .utils import file_exists, read_text


@dataclass(frozen=True)
class StoryKey:
    id: str
    prefix: str
    key: str


def sprint_status_file(project_root: str) -> str:
    preferred = Path(project_root) / "_bmad-output" / "implementation-artifacts" / "sprint-status.yaml"
    if preferred.is_file():
        return str(preferred)
    legacy = Path(project_root) / "_bmad-output" / "sprint-status.yaml"
    if legacy.is_file():
        return str(legacy)
    return str(preferred)


def normalize_story_key(project_root: str, value: str, state_file: str | Path | None = None) -> StoryKey | None:
    if re.fullmatch(r"\d+\.\d+", value):
        story_id = value
        prefix = value.replace(".", "-")
        key = ""
    elif re.fullmatch(r"\d+-\d+", value):
        prefix = value
        story_id = value.replace("-", ".")
        key = ""
    elif re.fullmatch(r"\d+-\d+-.+", value):
        key = value
        prefix = "-".join(value.split("-", 2)[:2])
        story_id = prefix.replace("-", ".")
    else:
        return None

    artifacts = Path(project_root) / "_bmad-output" / "implementation-artifacts"
    if not key:
        key = contextual_story_key(project_root, story_id, state_file=state_file)
    if not key:
        matches = sorted(artifacts.glob(f"{prefix}-*.md"))
        if matches:
            key = matches[0].stem
    if not key:
        status_file = sprint_status_file(project_root)
        if file_exists(status_file):
            content = read_text(status_file)
            match = re.search(rf"(?m)^\s*({re.escape(prefix)}-[^:\s]+)\s*:", content)
            if match:
                key = match.group(1).strip()
    if not key:
        key = prefix
    prefix = _story_prefix_from_key(key, prefix)
    return StoryKey(id=story_id, prefix=prefix, key=key)


def contextual_story_key(project_root: str, story_id: str, state_file: str | Path | None = None) -> str:
    context_slug = _context_product_slug(project_root, state_file=state_file)
    if not context_slug or not re.fullmatch(r"\d+\.\d+", story_id):
        return ""
    for key in contextual_epic_story_keys(project_root, story_id.split(".", 1)[0], state_file=state_file):
        if _story_id_from_key(key) == story_id:
            return key
    return ""


def contextual_epic_story_keys(project_root: str, epic_num: str, state_file: str | Path | None = None) -> list[str]:
    context_slug = _context_product_slug(project_root, state_file=state_file)
    if not context_slug:
        return []
    status_file = sprint_status_file(project_root)
    if not file_exists(status_file):
        return []
    return _story_keys_in_epic_block(read_text(status_file), context_slug, epic_num)


def _story_keys_in_epic_block(content: str, context_slug: str, epic_num: str) -> list[str]:
    lines = content.splitlines()
    epic_header = f"{context_slug}-epic-{epic_num}"
    keys: list[str] = []
    inside_block = False
    for raw_line in lines:
        line = raw_line.strip()
        if not line or line.startswith("#") or ":" not in line:
            continue
        key = line.split(":", 1)[0].strip()
        if not inside_block:
            inside_block = key == epic_header
            continue
        if _is_epic_marker(key):
            break
        if _story_id_from_key(key):
            keys.append(key)
    return keys


def _story_id_from_key(key: str) -> str:
    match = re.match(r"^(?:[a-z0-9]+-)*(\d+)-(\d+)-", key, re.IGNORECASE)
    if not match:
        return ""
    return f"{match.group(1)}.{match.group(2)}"


def _story_prefix_from_key(key: str, fallback: str) -> str:
    match = re.match(r"^((?:[a-z0-9]+-)*\d+-\d+)-.+$", key, re.IGNORECASE)
    if not match:
        return fallback
    return match.group(1)


def _is_epic_marker(key: str) -> bool:
    return bool(re.fullmatch(r"(?:[a-z0-9][a-z0-9-]*-)?epic-\d+(?:-retrospective)?", key, re.IGNORECASE))


def _context_product_slug(project_root: str, state_file: str | Path | None = None) -> str:
    fields = _load_context_state_fields(project_root, state_file=state_file)
    epic_source = str(fields.get("epicSource") or "").strip()
    if epic_source:
        match = re.fullmatch(r"epics-(.+)\.md", Path(epic_source).name, re.IGNORECASE)
        if match:
            return _slugify(match.group(1))
    return ""


def _load_context_state_fields(project_root: str, state_file: str | Path | None = None) -> dict[str, object]:
    path = _resolve_context_state_file(project_root, state_file=state_file)
    if path is None or not path.is_file():
        return {}
    try:
        return parse_simple_frontmatter(read_text(path))
    except OSError:
        return {}


def _resolve_context_state_file(project_root: str, state_file: str | Path | None = None) -> Path | None:
    root = Path(project_root).resolve()
    if state_file:
        return Path(state_file).expanduser().resolve()
    env_state = os.environ.get("STORY_AUTOMATOR_STATE_FILE", "").strip()
    if env_state:
        path = Path(env_state).expanduser()
        return (path if path.is_absolute() else (root / path)).resolve()
    marker = active_marker_path(root)
    if not marker.is_file():
        return None
    try:
        payload = json.loads(read_text(marker))
    except (OSError, json.JSONDecodeError):
        return None
    marker_state = str(payload.get("stateFile") or "").strip()
    if not marker_state:
        return None
    path = Path(marker_state).expanduser()
    return (path if path.is_absolute() else (root / path)).resolve()


def _slugify(value: str) -> str:
    slug = re.sub(r"[^a-z0-9]+", "-", value.strip().lower())
    return slug.strip("-")
