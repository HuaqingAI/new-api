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
    elif re.fullmatch(r"[A-Za-z0-9][\w-]*\.\d+", value):
        story_id = value
        epic_part, _, story_num = value.partition(".")
        prefix = f"{epic_part}-{story_num}"
        key = ""
    elif re.fullmatch(r"[A-Za-z0-9][\w-]*-\d+", value):
        contextual_story_id = _contextual_story_id_from_value(project_root, value, state_file=state_file)
        if contextual_story_id:
            story_id = contextual_story_id
            prefix = value
            key = contextual_story_key(project_root, story_id, state_file=state_file)
            return _complete_story_key(project_root, story_id, prefix, key, state_file=state_file)
        prefix = value
        epic_part, _, story_num = value.rpartition("-")
        story_id = f"{epic_part}.{story_num}"
        key = ""
    elif re.fullmatch(r"[A-Za-z0-9][\w-]*-\d+-.+", value):
        contextual_story_id = _contextual_story_id_from_value(project_root, value, state_file=state_file)
        if contextual_story_id:
            prefix = _story_prefix_from_key(value, value)
            return StoryKey(id=contextual_story_id, prefix=prefix, key=value)
        split = _split_non_numeric_full_key(project_root, value)
        if split is None:
            return None
        epic_part, story_num = split
        prefix = f"{epic_part}-{story_num}"
        story_id = f"{epic_part}.{story_num}"
        key = value
    else:
        return None

    return _complete_story_key(project_root, story_id, prefix, key, state_file=state_file)


def normalize_story_key_for_epic(
    project_root: str,
    epic: str,
    value: str,
    state_file: str | Path | None = None,
) -> StoryKey | None:
    if "." in value:
        norm = normalize_story_key(project_root, value, state_file=state_file)
        if norm is None or norm.id.rsplit(".", 1)[0].casefold() != epic.casefold():
            return None
        return norm

    dotted = re.fullmatch(rf"{re.escape(epic)}\.(\d+)", value, re.IGNORECASE)
    if dotted:
        story_num = dotted.group(1)
        return _complete_story_key_for_epic(project_root, epic, story_num, state_file=state_file)

    dashed = re.fullmatch(rf"{re.escape(epic)}-(\d+)(?:-.+)?", value, re.IGNORECASE)
    if dashed:
        if _has_known_longer_epic(project_root, epic, value) or _story_prefix_claimed_by_parent_epic(
            project_root,
            epic,
            value,
            state_file=state_file,
        ):
            return None
        story_num = dashed.group(1)
        norm = _complete_story_key_for_epic(project_root, epic, story_num, state_file=state_file)
        if norm is None:
            return None
        if value != norm.prefix and value.casefold() != norm.key.casefold():
            return StoryKey(id=norm.id, prefix=norm.prefix, key=value)
        return norm

    return normalize_story_key(project_root, value, state_file=state_file)


def contextual_story_key(project_root: str, story_id: str, state_file: str | Path | None = None) -> str:
    context_slug = _context_product_slug(project_root, state_file=state_file)
    if not context_slug:
        return ""
    for key in contextual_epic_story_keys(project_root, story_id.split(".", 1)[0], state_file=state_file):
        if _story_id_equal(_contextual_story_id_from_key(key), story_id):
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


def _complete_story_key(
    project_root: str,
    story_id: str,
    prefix: str,
    key: str,
    *,
    state_file: str | Path | None = None,
) -> StoryKey:
    artifacts = Path(project_root) / "_bmad-output" / "implementation-artifacts"
    if not key:
        key = contextual_story_key(project_root, story_id, state_file=state_file)
    if not key:
        for match in _artifact_matches_for_prefix(artifacts, prefix):
            if _full_key_matches_story(project_root, match.stem, story_id, allow_ambiguous_same_id=False):
                key = match.stem
                break
    if not key:
        status_file = sprint_status_file(project_root)
        if file_exists(status_file):
            status_key = _status_key_for_prefix(read_text(status_file), prefix, story_id)
            if status_key:
                key = status_key
    if not key:
        key = prefix
    prefix = _story_prefix_from_key(key, prefix)
    return StoryKey(id=story_id, prefix=prefix, key=key)


def _complete_story_key_for_epic(
    project_root: str,
    epic: str,
    story_num: str,
    *,
    state_file: str | Path | None = None,
) -> StoryKey | None:
    story_id = f"{epic}.{story_num}"
    prefix = f"{epic}-{story_num}"
    norm = normalize_story_key(project_root, story_id, state_file=state_file)
    if norm is not None:
        return norm
    key = _find_story_key_by_story_id(project_root, story_id)
    if not key:
        return StoryKey(id=story_id, prefix=prefix, key=prefix)
    return StoryKey(id=story_id, prefix=_story_prefix_from_key(key, prefix), key=key)


def _find_story_key_by_story_id(project_root: str, story_id: str) -> str:
    artifacts = Path(project_root) / "_bmad-output" / "implementation-artifacts"
    for match in sorted(artifacts.glob("*.md"), key=lambda path: path.name.casefold()):
        if _story_id_equal(_story_id_from_key(match.stem), story_id):
            return match.stem
    status_file = sprint_status_file(project_root)
    if not file_exists(status_file):
        return ""
    for status_key in _status_keys(read_text(status_file)):
        if _story_id_equal(_story_id_from_key(status_key), story_id):
            return status_key
    return ""


def _story_keys_in_epic_block(content: str, context_slug: str, epic_num: str) -> list[str]:
    lines = content.splitlines()
    epic_headers = {f"{context_slug}-epic-{epic_num}"}
    if context_slug == "agent-platform":
        epic_headers.add(f"ap-epic-{epic_num}")
    keys: list[str] = []
    inside_block = False
    for raw_line in lines:
        line = raw_line.strip()
        if not line or line.startswith("#") or ":" not in line:
            continue
        key = line.split(":", 1)[0].strip()
        if not inside_block:
            inside_block = any(key.casefold() == header.casefold() for header in epic_headers)
            continue
        if _is_epic_marker(key):
            break
        if _contextual_story_id_from_key(key):
            keys.append(key)
    return keys


def _story_id_from_key(key: str) -> str:
    split = _split_story_key_value(key)
    if split is None:
        return _contextual_story_id_from_key(key)
    epic_part, story_num = split
    return f"{epic_part}.{story_num}"


def _contextual_story_id_from_key(key: str) -> str:
    match = re.match(r"^(?:[a-z0-9]+-)*(\d+)-(\d+)-", key, re.IGNORECASE)
    if not match:
        return ""
    return f"{match.group(1)}.{match.group(2)}"


def _story_prefix_from_key(key: str, fallback: str) -> str:
    split = _split_story_key_value(key)
    if split is None:
        match = re.match(r"^((?:[a-z0-9]+-)*\d+-\d+)-.+$", key, re.IGNORECASE)
        return match.group(1) if match else fallback
    epic_part, story_num = split
    return f"{epic_part}-{story_num}"


def _contextual_story_id_from_value(project_root: str, value: str, state_file: str | Path | None = None) -> str:
    context_slug = _context_product_slug(project_root, state_file=state_file)
    if not context_slug:
        return ""
    match = re.match(r"^(?:[a-z0-9]+-)*(\d+)-(\d+)(?:-.+)?$", value, re.IGNORECASE)
    if not match:
        return ""
    story_id = f"{match.group(1)}.{match.group(2)}"
    epic_num = match.group(1)
    known_keys = contextual_epic_story_keys(project_root, epic_num, state_file=state_file)
    if not known_keys:
        return ""
    if any(value.casefold() == key.casefold() for key in known_keys):
        return story_id
    value_prefix = _story_prefix_from_key(value, value)
    if any(_story_prefix_from_key(key, key).casefold() == value_prefix.casefold() for key in known_keys):
        return story_id
    return ""


def _split_story_key_value(value: str) -> tuple[str, str] | None:
    if re.fullmatch(r"\d+-\d+-.+", value):
        left, right = value.split("-", 2)[:2]
        return left, right
    if re.fullmatch(r"[A-Za-z0-9][\w-]*-\d+-.+", value):
        return _split_non_numeric_full_key("", value)
    return None


def _status_key_for_prefix(content: str, prefix: str, story_id: str) -> str:
    for status_key in _status_keys(content):
        if status_key.casefold().startswith(f"{prefix.casefold()}-") and _full_key_matches_story(
            "",
            status_key,
            story_id,
            allow_ambiguous_same_id=True,
        ):
            return status_key
    return ""


def _is_epic_marker(key: str) -> bool:
    return bool(re.fullmatch(r"(?:[a-z0-9][a-z0-9-]*-)?epic-[\w-]+(?:-retrospective)?", key, re.IGNORECASE))


def _split_non_numeric_full_key(project_root: str, value: str) -> tuple[str, str] | None:
    matches = list(re.finditer(r"(?=-(\d+)-)", value))
    if not matches:
        return None
    single_story = [
        match
        for match in matches[1:]
        if _is_single_story_key(project_root, value, match)
    ]
    if single_story:
        match = max(single_story, key=lambda item: item.start())
        return value[: match.start()], match.group(1)
    known = [match for match in matches if _epic_exists(project_root, value[: match.start()])]
    if known:
        match = max(known, key=lambda item: item.start())
        return value[: match.start()], match.group(1)
    match = _numeric_epic_segment_match(matches) or _default_story_epic_match(value, matches)
    return value[: match.start()], match.group(1)


def _has_known_longer_epic(project_root: str, epic: str, value: str) -> bool:
    for match in re.finditer(r"(?=-(\d+)-)", value):
        candidate_epic = value[: match.start()]
        if candidate_epic.casefold() == epic.casefold() or not candidate_epic.casefold().startswith(f"{epic.casefold()}-"):
            continue
        if _epic_exists(project_root, candidate_epic) or _is_single_story_key(project_root, value, match):
            return True
    return False


def _status_keys(content: str) -> list[str]:
    keys: list[str] = []
    for line in content.splitlines():
        stripped = line.strip()
        if not stripped or stripped.startswith("#") or ":" not in stripped:
            continue
        keys.append(stripped.split(":", 1)[0].strip())
    return keys


def _full_key_matches_story(project_root: str, key: str, story_id: str, *, allow_ambiguous_same_id: bool) -> bool:
    norm = normalize_story_key(project_root, key) if project_root else _story_key_from_literal(key)
    if norm is not None:
        if not allow_ambiguous_same_id and _has_ambiguous_later_boundary(key, story_id):
            return False
        return _story_id_equal(norm.id, story_id)
    prefix = story_id.replace(".", "-")
    if not key.casefold().startswith(f"{prefix.casefold()}-"):
        return False
    story_num = story_id.rsplit(".", 1)[-1]
    remainder = key[len(prefix) + 1 :]
    first_segment = remainder.split("-", 1)[0]
    if not first_segment.isdigit():
        return not _has_ambiguous_later_boundary(key, story_id)
    return len(story_num) >= 4 and int(first_segment) <= 99


def _story_key_from_literal(value: str) -> StoryKey | None:
    if re.fullmatch(r"\d+-\d+-.+", value):
        prefix = "-".join(value.split("-", 2)[:2])
        return StoryKey(id=prefix.replace("-", "."), prefix=prefix, key=value)
    if re.fullmatch(r"[A-Za-z0-9][\w-]*-\d+-.+", value):
        split = _split_story_key_value(value)
        if split is None:
            return None
        epic_part, story_num = split
        prefix = f"{epic_part}-{story_num}"
        return StoryKey(id=f"{epic_part}.{story_num}", prefix=prefix, key=value)
    return None


def _has_ambiguous_later_boundary(key: str, story_id: str) -> bool:
    story_num = story_id.rsplit(".", 1)[-1]
    if len(story_num) < 4:
        return False
    prefix = story_id.replace(".", "-")
    if not key.casefold().startswith(f"{prefix.casefold()}-"):
        return False
    remainder = key[len(prefix) + 1 :]
    return re.search(r"^[^-]+-\d+-\d+-", remainder) is not None


def _epic_exists(project_root: str, epic: str) -> bool:
    if _epic_file_exists(project_root, epic):
        return True
    status_file = sprint_status_file(project_root)
    if not file_exists(status_file):
        return False
    pattern = re.compile(rf"(?m)^\s*{re.escape(epic)}-(\d+)(?:-[^:\s]+)?\s*:", re.IGNORECASE)
    story_nums = {match.group(1) for match in pattern.finditer(read_text(status_file))}
    return len(story_nums) > 1


def _is_single_story_key(project_root: str, value: str, match: re.Match[str]) -> bool:
    candidate_epic = value[: match.start()]
    story_num = match.group(1)
    if story_num != "1" and _known_parent_epic(project_root, candidate_epic):
        return False
    return (
        _has_exact_story_key(project_root, value)
        and _last_epic_segment_is_numeric(candidate_epic)
        and _single_story_numeric_epic(candidate_epic)
        and (
            story_num == "1"
            or (
                story_num.isdigit()
                and 10 <= int(story_num) <= 99
                and _year_like_numeric_suffix_epic(candidate_epic)
            )
        )
    )


def _default_story_epic_match(value: str, matches: list[re.Match[str]]) -> re.Match[str]:
    for match in reversed(matches[1:]):
        candidate_epic = value[: match.start()]
        story_num = match.group(1)
        if story_num.isdigit() and 10 <= int(story_num) <= 99 and _year_like_numeric_suffix_epic(candidate_epic):
            return match
    return matches[0]


def _numeric_epic_segment_match(matches: list[re.Match[str]]) -> re.Match[str] | None:
    if len(matches) < 2 or len(matches[0].group(1)) < 4 or len(matches[1].group(1)) < 3:
        return None
    return matches[1]


def _has_exact_story_key(project_root: str, value: str) -> bool:
    if not project_root:
        return False
    artifacts = Path(project_root) / "_bmad-output" / "implementation-artifacts"
    if (artifacts / f"{value}.md").is_file():
        return True
    status_file = sprint_status_file(project_root)
    return file_exists(status_file) and re.search(rf"(?m)^\s*{re.escape(value)}\s*:", read_text(status_file)) is not None


def _known_parent_epic(project_root: str, candidate_epic: str) -> bool:
    parent_epic, _, story_num = candidate_epic.rpartition("-")
    return bool(parent_epic and story_num.isdigit() and _epic_exists(project_root, parent_epic))


def _story_prefix_claimed_by_parent_epic(project_root: str, epic: str, value: str, state_file: str | Path | None = None) -> bool:
    parent_epic, _, story_num = epic.rpartition("-")
    return bool(
        parent_epic
        and story_num.isdigit()
        and not _epic_exists(project_root, epic)
        and normalize_story_key_for_epic(project_root, parent_epic, value, state_file=state_file) is not None
    )


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


def _last_epic_segment_is_numeric(epic: str) -> bool:
    return epic.rsplit("-", 1)[-1].isdigit()


def _year_like_numeric_suffix_epic(epic: str) -> bool:
    segment = epic.rsplit("-", 1)[-1]
    return segment.isdigit() and int(segment) >= 1000


def _single_story_numeric_epic(epic: str) -> bool:
    segments = epic.split("-")
    numeric_indexes = [index for index, segment in enumerate(segments) if segment.isdigit()]
    if len(numeric_indexes) == 1:
        return True
    if len(numeric_indexes) == 2:
        first, second = numeric_indexes
        return int(segments[second]) <= 99 and any(not segment.isdigit() for segment in segments[first + 1 : second])
    return False


def _epic_file_exists(project_root: str, epic: str) -> bool:
    root = Path(project_root)
    for base in (root / "_bmad-output" / "implementation-artifacts", root / "docs" / "epics"):
        if (base / f"epic-{epic}.md").is_file() or next(base.glob(f"epic-{epic}-*.md"), None) is not None:
            return True
    return False


def _artifact_matches_for_prefix(artifacts: Path, prefix: str) -> list[Path]:
    needle = f"{prefix.casefold()}-"
    return sorted(
        (path for path in artifacts.glob("*.md") if path.stem.casefold().startswith(needle)),
        key=lambda path: path.name.casefold(),
    )


def _story_id_equal(left: str, right: str) -> bool:
    return left.casefold() == right.casefold()


def _slugify(value: str) -> str:
    slug = re.sub(r"[^a-z0-9]+", "-", value.strip().lower())
    return slug.strip("-")
