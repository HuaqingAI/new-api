#!/usr/bin/env python3
"""Image generation/editing CLI backed by a new-api OpenAI-compatible endpoint."""

from __future__ import annotations

import argparse
import asyncio
import base64
from dataclasses import dataclass
from datetime import datetime
import json
import mimetypes
import os
from pathlib import Path
import re
import socket
import sys
import time
from typing import Any, Dict, Iterable, List, Optional, Tuple
from urllib import error as urlerror
from urllib import parse as urlparse
from urllib import request as urlrequest
import uuid

from io import BytesIO

FIXED_MODEL = "gpt-image-2"
DEFAULT_SIZE = "auto"
DEFAULT_QUALITY = "medium"
DEFAULT_OUTPUT_FORMAT = "png"
DEFAULT_RESPONSE_FORMAT = "b64_json"
DEFAULT_CONCURRENCY = 5
DEFAULT_DOWNSCALE_SUFFIX = "-web"
DEFAULT_OUTPUT_DIR = "output/imagegen"
DEFAULT_STATE_FILE = "output/imagegen/.imagegen-state.json"

ALLOWED_QUALITIES = {"low", "medium", "high", "auto"}
ALLOWED_BACKGROUNDS = {"transparent", "opaque", "auto", None}
ALLOWED_INPUT_FIDELITIES = {"low", "high", None}
ALLOWED_RESPONSE_FORMATS = {"b64_json", "url", None}

GPT_IMAGE_2_MIN_PIXELS = 655_360
GPT_IMAGE_2_MAX_PIXELS = 8_294_400
GPT_IMAGE_2_MAX_EDGE = 3840
GPT_IMAGE_2_MAX_RATIO = 3.0

MAX_IMAGE_BYTES = 50 * 1024 * 1024
MAX_BATCH_JOBS = 500
MAX_STATE_HISTORY = 100


@dataclass
class APIConfig:
    base_url: str
    api_key: str
    provider: str = ""
    base_url_source: str = ""
    api_key_source: str = ""


@dataclass
class PartialConfig:
    base_url: str = ""
    api_key: str = ""
    provider: str = ""
    base_url_source: str = ""
    api_key_source: str = ""


class ImageAPIError(RuntimeError):
    def __init__(self, status: int, body: str, headers: Any = None):
        self.status = status
        self.body = body
        self.headers = headers
        super().__init__(f"HTTP {status}: {_extract_error_message(body)}")


def _die(message: str, code: int = 1) -> None:
    print(f"Error: {message}", file=sys.stderr)
    raise SystemExit(code)


def _warn(message: str) -> None:
    print(f"Warning: {message}", file=sys.stderr)


def _read_prompt(prompt: Optional[str], prompt_file: Optional[str]) -> str:
    if prompt and prompt_file:
        _die("Use --prompt or --prompt-file, not both.")
    if prompt_file:
        path = Path(prompt_file)
        if not path.exists():
            _die(f"Prompt file not found: {path}")
        return path.read_text(encoding="utf-8").strip()
    if prompt:
        return prompt.strip()
    _die("Missing prompt. Use --prompt or --prompt-file.")
    return ""


def _resolve_workspace(args: argparse.Namespace) -> Path:
    raw = args.workspace or os.getenv("AIONUI_WORKSPACE") or os.getcwd()
    return Path(raw).expanduser().resolve()


def _resolve_path(raw: str, workspace: Path) -> Path:
    path = Path(raw).expanduser()
    if not path.is_absolute():
        path = workspace / path
    return path.resolve()


def _relative_to_workspace(path: Path, workspace: Path) -> str:
    try:
        return path.resolve().relative_to(workspace.resolve()).as_posix()
    except ValueError:
        return str(path.resolve())


def _dedupe_paths(paths: Iterable[Path]) -> List[Path]:
    seen = set()
    result: List[Path] = []
    for path in paths:
        key = str(path.expanduser())
        if key in seen:
            continue
        seen.add(key)
        result.append(path.expanduser())
    return result


def _normalize_base_url(value: str) -> str:
    raw = value.strip().rstrip("/")
    if not raw:
        return ""

    parsed = urlparse.urlparse(raw)
    if not parsed.scheme or not parsed.netloc:
        _die(f"Invalid base URL: {value}")

    path = re.sub(r"/{2,}", "/", parsed.path or "")
    for suffix in ("/images/generations", "/images/edits"):
        if path.endswith(suffix):
            path = path[: -len(suffix)]
    path = path.rstrip("/")
    if not path.endswith("/v1"):
        path = f"{path}/v1" if path else "/v1"

    return urlparse.urlunparse((parsed.scheme, parsed.netloc, path, "", "", ""))


def _strip_jsonc_comments(text: str) -> str:
    out: List[str] = []
    i = 0
    in_string = False
    escape = False
    in_line_comment = False
    in_block_comment = False
    while i < len(text):
        ch = text[i]
        nxt = text[i + 1] if i + 1 < len(text) else ""

        if in_line_comment:
            if ch in "\r\n":
                in_line_comment = False
                out.append(ch)
            i += 1
            continue

        if in_block_comment:
            if ch == "*" and nxt == "/":
                in_block_comment = False
                i += 2
            else:
                if ch in "\r\n":
                    out.append(ch)
                i += 1
            continue

        if in_string:
            out.append(ch)
            if escape:
                escape = False
            elif ch == "\\":
                escape = True
            elif ch == '"':
                in_string = False
            i += 1
            continue

        if ch == '"':
            in_string = True
            out.append(ch)
            i += 1
            continue

        if ch == "/" and nxt == "/":
            in_line_comment = True
            i += 2
            continue

        if ch == "/" and nxt == "*":
            in_block_comment = True
            i += 2
            continue

        out.append(ch)
        i += 1

    return "".join(out)


def _load_jsonc(path: Path) -> Dict[str, Any]:
    text = path.read_text(encoding="utf-8-sig")
    text = _strip_jsonc_comments(text)
    text = re.sub(r",\s*([}\]])", r"\1", text)
    data = json.loads(text)
    return data if isinstance(data, dict) else {}


def _load_toml(path: Path) -> Dict[str, Any]:
    try:
        import tomllib

        with path.open("rb") as handle:
            data = tomllib.load(handle)
        return data if isinstance(data, dict) else {}
    except ModuleNotFoundError:
        return _load_simple_toml(path)


def _load_simple_toml(path: Path) -> Dict[str, Any]:
    root: Dict[str, Any] = {}
    current: Dict[str, Any] = root
    for raw_line in path.read_text(encoding="utf-8-sig").splitlines():
        line = raw_line.split("#", 1)[0].strip()
        if not line:
            continue
        if line.startswith("[") and line.endswith("]"):
            current = root
            for part in line.strip("[]").split("."):
                current = current.setdefault(part, {})
            continue
        if "=" not in line:
            continue
        key, value = line.split("=", 1)
        key = key.strip()
        value = value.strip().strip('"').strip("'")
        current[key] = value
    return root


def _candidate_opencode_paths(workspace: Path) -> List[Path]:
    candidates: List[Path] = []
    config_dir = os.getenv("OPENCODE_CONFIG_DIR")
    if config_dir:
        p = Path(config_dir).expanduser()
        candidates.append(p if p.suffix else p / "opencode.jsonc")
    candidates.extend(
        [
            workspace / "opencode.jsonc",
            workspace / ".opencode" / "opencode.jsonc",
        ]
    )
    appdata = os.getenv("APPDATA")
    if appdata:
        candidates.append(Path(appdata) / "opencode" / "opencode.jsonc")
    return _dedupe_paths(candidates)


def _candidate_codex_config_paths(workspace: Path) -> List[Path]:
    candidates = [workspace / ".codex" / "config.toml"]
    codex_home = os.getenv("CODEX_HOME")
    if codex_home:
        candidates.append(Path(codex_home).expanduser() / "config.toml")
    candidates.append(Path.home() / ".codex" / "config.toml")
    return _dedupe_paths(candidates)


def _candidate_codex_auth_paths(workspace: Path) -> List[Path]:
    candidates = [workspace / ".codex" / "auth.json"]
    codex_home = os.getenv("CODEX_HOME")
    if codex_home:
        candidates.append(Path(codex_home).expanduser() / "auth.json")
    candidates.append(Path.home() / ".codex" / "auth.json")
    return _dedupe_paths(candidates)


def _provider_from_model(model: Any) -> str:
    if not isinstance(model, str) or "/" not in model:
        return ""
    provider, _ = model.split("/", 1)
    return provider.strip()


def _provider_config_from_map(
    providers: Any,
    provider_hint: str,
    default_provider: str = "hth",
) -> Tuple[str, Dict[str, Any]]:
    if not isinstance(providers, dict) or not providers:
        return "", {}
    if provider_hint and isinstance(providers.get(provider_hint), dict):
        return provider_hint, providers[provider_hint]
    if default_provider in providers and isinstance(providers[default_provider], dict):
        return default_provider, providers[default_provider]
    for name, cfg in providers.items():
        if isinstance(cfg, dict):
            return str(name), cfg
    return "", {}


def _read_opencode_config(path: Path, provider_hint: str = "") -> PartialConfig:
    data = _load_jsonc(path)
    providers = data.get("provider") or data.get("providers") or {}
    provider_name = provider_hint or _provider_from_model(data.get("model"))
    provider_name, provider_cfg = _provider_config_from_map(providers, provider_name)
    if not provider_cfg:
        return PartialConfig(provider=provider_name)

    options = provider_cfg.get("options") if isinstance(provider_cfg.get("options"), dict) else {}
    base_url = (
        provider_cfg.get("api")
        or provider_cfg.get("base_url")
        or provider_cfg.get("baseURL")
        or ""
    )
    api_key = (
        options.get("apiKey")
        or options.get("api_key")
        or provider_cfg.get("apiKey")
        or provider_cfg.get("api_key")
        or ""
    )
    return PartialConfig(
        base_url=str(base_url).strip(),
        api_key=str(api_key).strip(),
        provider=provider_name,
        base_url_source=f"{path}:provider.{provider_name}.api" if base_url else "",
        api_key_source=f"{path}:provider.{provider_name}.options.apiKey" if api_key else "",
    )


def _read_codex_config(path: Path, provider_hint: str = "") -> PartialConfig:
    data = _load_toml(path)
    provider_name = provider_hint or str(data.get("model_provider") or "").strip()
    providers = data.get("model_providers") or {}
    provider_name, provider_cfg = _provider_config_from_map(providers, provider_name)
    if not provider_cfg:
        return PartialConfig(provider=provider_name)

    base_url = provider_cfg.get("base_url") or provider_cfg.get("baseURL") or ""
    return PartialConfig(
        base_url=str(base_url).strip(),
        provider=provider_name,
        base_url_source=f"{path}:model_providers.{provider_name}.base_url" if base_url else "",
    )


def _read_codex_auth(path: Path) -> PartialConfig:
    data = _load_jsonc(path)
    api_key = data.get("OPENAI_API_KEY") or data.get("api_key") or ""
    return PartialConfig(
        api_key=str(api_key).strip(),
        api_key_source=f"{path}:OPENAI_API_KEY" if api_key else "",
    )


def _merge_partial(left: PartialConfig, right: PartialConfig) -> PartialConfig:
    return PartialConfig(
        base_url=left.base_url or right.base_url,
        api_key=left.api_key or right.api_key,
        provider=left.provider or right.provider,
        base_url_source=left.base_url_source or right.base_url_source,
        api_key_source=left.api_key_source or right.api_key_source,
    )


def _discover_file_config(workspace: Path, provider_hint: str) -> PartialConfig:
    found = PartialConfig(provider=provider_hint)

    for path in _candidate_opencode_paths(workspace):
        if not path.exists():
            continue
        try:
            cfg = _read_opencode_config(path, provider_hint)
        except Exception as exc:
            _warn(f"Failed to read OpenCode config {path}: {exc}")
            continue
        found = _merge_partial(found, cfg)
        if found.base_url and found.api_key:
            return found

    for path in _candidate_codex_config_paths(workspace):
        if not path.exists():
            continue
        try:
            cfg = _read_codex_config(path, provider_hint or found.provider)
        except Exception as exc:
            _warn(f"Failed to read Codex config {path}: {exc}")
            continue
        found = _merge_partial(found, cfg)

    for path in _candidate_codex_auth_paths(workspace):
        if not path.exists():
            continue
        try:
            cfg = _read_codex_auth(path)
        except Exception as exc:
            _warn(f"Failed to read Codex auth {path}: {exc}")
            continue
        found = _merge_partial(found, cfg)
        if found.base_url and found.api_key:
            return found

    return found


def _apply_override(
    cfg: PartialConfig,
    *,
    base_url: Optional[str] = None,
    api_key: Optional[str] = None,
    provider: Optional[str] = None,
    source: str,
) -> PartialConfig:
    if base_url:
        cfg.base_url = base_url.strip()
        cfg.base_url_source = source
    if api_key:
        cfg.api_key = api_key.strip()
        cfg.api_key_source = source
    if provider:
        cfg.provider = provider.strip()
    return cfg


def _resolve_api_config(args: argparse.Namespace, workspace: Path) -> APIConfig:
    env_provider = os.getenv("IMAGEGEN_PROVIDER", "").strip()
    provider_hint = (args.provider or env_provider).strip()
    cfg = _discover_file_config(workspace, provider_hint)

    cfg = _apply_override(
        cfg,
        base_url=os.getenv("OPENAI_API_BASE") or None,
        source="OPENAI_API_BASE",
    )
    cfg = _apply_override(
        cfg,
        base_url=os.getenv("OPENAI_BASE_URL") or None,
        api_key=os.getenv("OPENAI_API_KEY") or None,
        source="OPENAI_* environment",
    )
    cfg = _apply_override(
        cfg,
        base_url=os.getenv("IMAGEGEN_BASE_URL") or None,
        api_key=os.getenv("IMAGEGEN_API_KEY") or None,
        provider=env_provider or None,
        source="IMAGEGEN_* environment",
    )
    cfg = _apply_override(
        cfg,
        base_url=args.base_url,
        api_key=args.api_key,
        provider=args.provider,
        source="CLI argument",
    )

    if not cfg.base_url:
        if args.dry_run:
            cfg.base_url = "https://example.invalid/v1"
            cfg.base_url_source = "dry-run placeholder"
        else:
            _die("API base URL not found. Configure OpenCode/Codex, IMAGEGEN_BASE_URL, OPENAI_BASE_URL, or --base-url.")
    if not cfg.api_key:
        if args.dry_run:
            cfg.api_key_source = "missing (dry-run)"
        else:
            _die("API key not found. Configure OpenCode/Codex auth, IMAGEGEN_API_KEY, OPENAI_API_KEY, or --api-key.")

    return APIConfig(
        base_url=_normalize_base_url(cfg.base_url),
        api_key=cfg.api_key,
        provider=cfg.provider,
        base_url_source=cfg.base_url_source,
        api_key_source=cfg.api_key_source,
    )


def _normalize_output_format(fmt: Optional[str]) -> str:
    if not fmt:
        return DEFAULT_OUTPUT_FORMAT
    fmt = fmt.lower()
    if fmt not in {"png", "jpeg", "jpg", "webp"}:
        _die("output-format must be png, jpeg, jpg, or webp.")
    return "jpeg" if fmt == "jpg" else fmt


def _parse_size(size: str) -> Optional[Tuple[int, int]]:
    match = re.fullmatch(r"([1-9][0-9]*)x([1-9][0-9]*)", size)
    if not match:
        return None
    return int(match.group(1)), int(match.group(2))


def _validate_gpt_image_2_size(size: str) -> None:
    if size == "auto":
        return

    parsed = _parse_size(size)
    if parsed is None:
        _die("size must be auto or WIDTHxHEIGHT, for example 1024x1024.")

    width, height = parsed
    max_edge = max(width, height)
    min_edge = min(width, height)
    total_pixels = width * height

    if max_edge > GPT_IMAGE_2_MAX_EDGE:
        _die("gpt-image-2 size maximum edge length must be less than or equal to 3840px.")
    if width % 16 != 0 or height % 16 != 0:
        _die("gpt-image-2 size width and height must be multiples of 16px.")
    if max_edge / min_edge > GPT_IMAGE_2_MAX_RATIO:
        _die("gpt-image-2 size long edge to short edge ratio must not exceed 3:1.")
    if total_pixels < GPT_IMAGE_2_MIN_PIXELS or total_pixels > GPT_IMAGE_2_MAX_PIXELS:
        _die("gpt-image-2 size total pixels must be at least 655,360 and no more than 8,294,400.")


def _validate_quality(quality: str) -> None:
    if quality not in ALLOWED_QUALITIES:
        _die("quality must be one of low, medium, high, or auto.")


def _validate_background(background: Optional[str]) -> None:
    if background not in ALLOWED_BACKGROUNDS:
        _die("background must be one of transparent, opaque, or auto.")


def _validate_input_fidelity(input_fidelity: Optional[str]) -> None:
    if input_fidelity not in ALLOWED_INPUT_FIDELITIES:
        _die("input-fidelity must be one of low or high.")


def _validate_response_format(response_format: Optional[str]) -> None:
    if response_format not in ALLOWED_RESPONSE_FORMATS:
        _die("response-format must be b64_json or url.")


def _validate_transparency(background: Optional[str], output_format: str) -> None:
    if background == "transparent" and output_format not in {"png", "webp"}:
        _die("transparent background requires output-format png or webp.")


def _validate_model_specific_options(background: Optional[str], input_fidelity: Optional[str] = None) -> None:
    if background == "transparent":
        _die("background=transparent is not available because this skill is fixed to gpt-image-2.")
    if input_fidelity is not None:
        _die("input_fidelity is not supported because this skill is fixed to gpt-image-2.")


def _validate_legacy_model_arg(args: argparse.Namespace) -> None:
    model = getattr(args, "legacy_model", None)
    if model and model != FIXED_MODEL:
        _die(f"model is fixed to {FIXED_MODEL}; remove --model or use --model {FIXED_MODEL}.")


def _validate_payload_options(payload: Dict[str, Any], *, input_fidelity: Optional[str] = None) -> None:
    n = int(payload.get("n", 1))
    if n < 1 or n > 10:
        _die("n must be between 1 and 10")
    _validate_gpt_image_2_size(str(payload.get("size", DEFAULT_SIZE)))
    _validate_quality(str(payload.get("quality", DEFAULT_QUALITY)))
    _validate_background(payload.get("background"))
    _validate_response_format(payload.get("response_format"))
    _validate_model_specific_options(payload.get("background"), input_fidelity)
    output_format = _normalize_output_format(payload.get("output_format"))
    _validate_transparency(payload.get("background"), output_format)
    oc = payload.get("output_compression")
    if oc is not None and not (0 <= int(oc) <= 100):
        _die("output_compression must be between 0 and 100")


def _build_output_paths(
    *,
    out: Optional[str],
    output_format: str,
    count: int,
    out_dir: Optional[str],
    prompt: str,
    workspace: Path,
) -> List[Path]:
    ext = "." + output_format

    if out_dir:
        out_base = _resolve_path(out_dir, workspace)
        out_base.mkdir(parents=True, exist_ok=True)
        return [out_base / f"image_{i}{ext}" for i in range(1, count + 1)]

    if out:
        out_path = _resolve_path(out, workspace)
    else:
        slug = _slugify(prompt[:80])
        stamp = datetime.now().strftime("%Y%m%d-%H%M%S")
        out_path = workspace / DEFAULT_OUTPUT_DIR / f"{stamp}-{slug}{ext}"

    if out_path.exists() and out_path.is_dir():
        out_path.mkdir(parents=True, exist_ok=True)
        return [out_path / f"image_{i}{ext}" for i in range(1, count + 1)]

    if out_path.suffix == "":
        out_path = out_path.with_suffix(ext)
    elif output_format and out_path.suffix.lstrip(".").lower() != output_format:
        _warn(f"Output extension {out_path.suffix} does not match output-format {output_format}.")

    if count == 1:
        return [out_path]

    return [out_path.with_name(f"{out_path.stem}-{i}{out_path.suffix}") for i in range(1, count + 1)]


def _augment_prompt(args: argparse.Namespace, prompt: str) -> str:
    fields = _fields_from_args(args)
    return _augment_prompt_fields(args.augment, prompt, fields)


def _augment_prompt_fields(augment: bool, prompt: str, fields: Dict[str, Optional[str]]) -> str:
    if not augment:
        return prompt

    sections: List[str] = []
    if fields.get("use_case"):
        sections.append(f"Use case: {fields['use_case']}")
    sections.append(f"Primary request: {prompt}")
    if fields.get("scene"):
        sections.append(f"Scene/background: {fields['scene']}")
    if fields.get("subject"):
        sections.append(f"Subject: {fields['subject']}")
    if fields.get("style"):
        sections.append(f"Style/medium: {fields['style']}")
    if fields.get("composition"):
        sections.append(f"Composition/framing: {fields['composition']}")
    if fields.get("lighting"):
        sections.append(f"Lighting/mood: {fields['lighting']}")
    if fields.get("palette"):
        sections.append(f"Color palette: {fields['palette']}")
    if fields.get("materials"):
        sections.append(f"Materials/textures: {fields['materials']}")
    if fields.get("text"):
        sections.append(f"Text (verbatim): \"{fields['text']}\"")
    if fields.get("constraints"):
        sections.append(f"Constraints: {fields['constraints']}")
    if fields.get("negative"):
        sections.append(f"Avoid: {fields['negative']}")

    return "\n".join(sections)


def _fields_from_args(args: argparse.Namespace) -> Dict[str, Optional[str]]:
    return {
        "use_case": getattr(args, "use_case", None),
        "scene": getattr(args, "scene", None),
        "subject": getattr(args, "subject", None),
        "style": getattr(args, "style", None),
        "composition": getattr(args, "composition", None),
        "lighting": getattr(args, "lighting", None),
        "palette": getattr(args, "palette", None),
        "materials": getattr(args, "materials", None),
        "text": getattr(args, "text", None),
        "constraints": getattr(args, "constraints", None),
        "negative": getattr(args, "negative", None),
    }


def _derive_downscale_path(path: Path, suffix: str) -> Path:
    if suffix and not suffix.startswith("-") and not suffix.startswith("_"):
        suffix = "-" + suffix
    return path.with_name(f"{path.stem}{suffix}{path.suffix}")


def _downscale_image_bytes(image_bytes: bytes, *, max_dim: int, output_format: str) -> bytes:
    try:
        from PIL import Image
    except Exception:
        _die("Downscaling requires Pillow. Install it with `uv pip install pillow`.")

    if max_dim < 1:
        _die("--downscale-max-dim must be >= 1")

    with Image.open(BytesIO(image_bytes)) as img:
        img.load()
        width, height = img.size
        scale = min(1.0, float(max_dim) / float(max(width, height)))
        target = (max(1, int(round(width * scale))), max(1, int(round(height * scale))))
        resized = img if target == (width, height) else img.resize(target, Image.Resampling.LANCZOS)

        fmt = output_format.lower()
        if fmt == "jpg":
            fmt = "jpeg"

        if fmt == "jpeg":
            if resized.mode in ("RGBA", "LA") or ("transparency" in getattr(resized, "info", {})):
                bg = Image.new("RGB", resized.size, (255, 255, 255))
                bg.paste(resized.convert("RGBA"), mask=resized.convert("RGBA").split()[-1])
                resized = bg
            else:
                resized = resized.convert("RGB")

        out = BytesIO()
        resized.save(out, format=fmt.upper())
        return out.getvalue()


def _slugify(value: str) -> str:
    value = value.strip().lower()
    value = re.sub(r"[^a-z0-9]+", "-", value)
    value = re.sub(r"-{2,}", "-", value).strip("-")
    return value[:60] if value else "image"


def _normalize_job(job: Any, idx: int) -> Dict[str, Any]:
    if isinstance(job, str):
        prompt = job.strip()
        if not prompt:
            _die(f"Empty prompt at job {idx}")
        return {"prompt": prompt}
    if isinstance(job, dict):
        if "prompt" not in job or not str(job["prompt"]).strip():
            _die(f"Missing prompt for job {idx}")
        if "model" in job and str(job["model"]).strip() not in {"", FIXED_MODEL}:
            _die(f"Job {idx}: model is fixed to {FIXED_MODEL}")
        return job
    _die(f"Invalid job at index {idx}: expected string or object.")
    return {}


def _read_jobs_jsonl(path: str) -> List[Dict[str, Any]]:
    p = Path(path)
    if not p.exists():
        _die(f"Input file not found: {p}")
    jobs: List[Dict[str, Any]] = []
    for line_no, raw in enumerate(p.read_text(encoding="utf-8").splitlines(), start=1):
        line = raw.strip()
        if not line or line.startswith("#"):
            continue
        try:
            item: Any
            if line.startswith("{"):
                item = json.loads(line)
            else:
                item = line
            jobs.append(_normalize_job(item, idx=line_no))
        except json.JSONDecodeError as exc:
            _die(f"Invalid JSON on line {line_no}: {exc}")
    if not jobs:
        _die("No jobs found in input file.")
    if len(jobs) > MAX_BATCH_JOBS:
        _die(f"Too many jobs ({len(jobs)}). Max is {MAX_BATCH_JOBS}.")
    return jobs


def _merge_non_null(dst: Dict[str, Any], src: Dict[str, Any]) -> Dict[str, Any]:
    merged = dict(dst)
    for key, value in src.items():
        if value is not None and key != "model":
            merged[key] = value
    return merged


def _job_output_paths(
    *,
    out_dir: Path,
    output_format: str,
    idx: int,
    prompt: str,
    n: int,
    explicit_out: Optional[str],
) -> List[Path]:
    out_dir.mkdir(parents=True, exist_ok=True)
    ext = "." + output_format

    if explicit_out:
        base = Path(explicit_out)
        if base.suffix == "":
            base = base.with_suffix(ext)
        elif base.suffix.lstrip(".").lower() != output_format:
            _warn(f"Job {idx}: output extension {base.suffix} does not match output-format {output_format}.")
        base = out_dir / base.name
    else:
        slug = _slugify(prompt[:80])
        base = out_dir / f"{idx:03d}-{slug}{ext}"

    if n == 1:
        return [base]
    return [base.with_name(f"{base.stem}-{i}{base.suffix}") for i in range(1, n + 1)]


def _extract_error_message(body: str) -> str:
    if not body:
        return "empty response body"
    try:
        data = json.loads(body)
    except Exception:
        return body[:800]
    err = data.get("error") if isinstance(data, dict) else None
    if isinstance(err, dict):
        return str(err.get("message") or err.get("type") or err)[:800]
    return str(data)[:800]


def _retry_after_seconds(headers: Any) -> Optional[float]:
    if not headers:
        return None
    try:
        value = headers.get("Retry-After")
    except Exception:
        value = None
    if not value:
        return None
    try:
        return max(0.0, float(value))
    except Exception:
        return None


def _should_retry_status(status: int) -> bool:
    return status == 429 or 500 <= status <= 599


def _request_with_retries(send, *, attempts: int) -> bytes:
    attempts = max(1, attempts)
    for attempt in range(1, attempts + 1):
        try:
            return send()
        except ImageAPIError as exc:
            if attempt >= attempts or not _should_retry_status(exc.status):
                raise
            sleep_s = _retry_after_seconds(exc.headers)
            if sleep_s is None:
                sleep_s = min(60.0, 2.0 ** attempt)
            _warn(f"Request attempt {attempt}/{attempts} failed with HTTP {exc.status}; retrying in {sleep_s:.1f}s")
            time.sleep(sleep_s)
        except (urlerror.URLError, TimeoutError, socket.timeout) as exc:
            if attempt >= attempts:
                raise RuntimeError(f"network error: {exc}") from exc
            sleep_s = min(60.0, 2.0 ** attempt)
            _warn(f"Request attempt {attempt}/{attempts} failed ({exc}); retrying in {sleep_s:.1f}s")
            time.sleep(sleep_s)
    raise RuntimeError("request failed")


def _http_post(url: str, *, headers: Dict[str, str], body: bytes, timeout: int) -> bytes:
    req = urlrequest.Request(url, data=body, headers=headers, method="POST")
    try:
        with urlrequest.urlopen(req, timeout=timeout) as resp:
            return resp.read()
    except urlerror.HTTPError as exc:
        error_body = exc.read().decode("utf-8", errors="replace")
        raise ImageAPIError(exc.code, error_body, exc.headers) from exc


def _post_json(auth: APIConfig, path: str, payload: Dict[str, Any], *, timeout: int, attempts: int) -> Dict[str, Any]:
    url = auth.base_url.rstrip("/") + path
    headers = {
        "Authorization": f"Bearer {auth.api_key}",
        "Content-Type": "application/json",
        "Accept": "application/json",
    }

    def send_payload(data: Dict[str, Any]) -> bytes:
        body = json.dumps(data, ensure_ascii=False, separators=(",", ":")).encode("utf-8")
        return _http_post(url, headers=headers, body=body, timeout=timeout)

    try:
        raw = _request_with_retries(lambda: send_payload(payload), attempts=attempts)
    except ImageAPIError as exc:
        if "response_format" in payload and "response_format" in exc.body.lower():
            _warn("Upstream rejected response_format; retrying once without it.")
            retry_payload = dict(payload)
            retry_payload.pop("response_format", None)
            raw = _request_with_retries(lambda: send_payload(retry_payload), attempts=attempts)
        else:
            raise

    return _decode_response_json(raw)


def _post_multipart(
    auth: APIConfig,
    path: str,
    fields: Dict[str, Any],
    files: List[Tuple[str, Path]],
    *,
    timeout: int,
    attempts: int,
) -> Dict[str, Any]:
    url = auth.base_url.rstrip("/") + path

    def send_fields(data: Dict[str, Any]) -> bytes:
        body, content_type = _build_multipart_body(data, files)
        headers = {
            "Authorization": f"Bearer {auth.api_key}",
            "Content-Type": content_type,
            "Accept": "application/json",
        }
        return _http_post(url, headers=headers, body=body, timeout=timeout)

    try:
        raw = _request_with_retries(lambda: send_fields(fields), attempts=attempts)
    except ImageAPIError as exc:
        if "response_format" in fields and "response_format" in exc.body.lower():
            _warn("Upstream rejected response_format; retrying once without it.")
            retry_fields = dict(fields)
            retry_fields.pop("response_format", None)
            raw = _request_with_retries(lambda: send_fields(retry_fields), attempts=attempts)
        else:
            raise

    return _decode_response_json(raw)


def _build_multipart_body(fields: Dict[str, Any], files: List[Tuple[str, Path]]) -> Tuple[bytes, str]:
    boundary = f"----imagegen-{uuid.uuid4().hex}"
    chunks: List[bytes] = []

    for name, value in fields.items():
        if value is None:
            continue
        chunks.append(f"--{boundary}\r\n".encode("utf-8"))
        chunks.append(f'Content-Disposition: form-data; name="{name}"\r\n\r\n'.encode("utf-8"))
        chunks.append(_form_value(value).encode("utf-8"))
        chunks.append(b"\r\n")

    for field_name, path in files:
        mime = _mime_type_from_path(path)
        chunks.append(f"--{boundary}\r\n".encode("utf-8"))
        chunks.append(
            (
                f'Content-Disposition: form-data; name="{field_name}"; filename="{path.name}"\r\n'
                f"Content-Type: {mime}\r\n\r\n"
            ).encode("utf-8")
        )
        chunks.append(path.read_bytes())
        chunks.append(b"\r\n")

    chunks.append(f"--{boundary}--\r\n".encode("utf-8"))
    return b"".join(chunks), f"multipart/form-data; boundary={boundary}"


def _form_value(value: Any) -> str:
    if isinstance(value, bool):
        return "true" if value else "false"
    return str(value)


def _decode_response_json(raw: bytes) -> Dict[str, Any]:
    try:
        data = json.loads(raw.decode("utf-8"))
    except Exception as exc:
        preview = raw[:800].decode("utf-8", errors="replace")
        raise RuntimeError(f"Image API returned non-JSON response: {preview}") from exc
    if not isinstance(data, dict):
        raise RuntimeError("Image API returned a non-object JSON response")
    if isinstance(data.get("error"), dict):
        raise RuntimeError(_extract_error_message(json.dumps(data, ensure_ascii=False)))
    return data


def _download_image_url(url: str, auth: APIConfig, *, timeout: int, attempts: int) -> bytes:
    base = urlparse.urlparse(auth.base_url)
    target = urlparse.urlparse(url)
    headers = {"Accept": "image/*"}
    if target.scheme == base.scheme and target.netloc == base.netloc and auth.api_key:
        headers["Authorization"] = f"Bearer {auth.api_key}"
    req = urlrequest.Request(url, headers=headers, method="GET")

    def send() -> bytes:
        try:
            with urlrequest.urlopen(req, timeout=timeout) as resp:
                return resp.read()
        except urlerror.HTTPError as exc:
            body = exc.read().decode("utf-8", errors="replace")
            raise ImageAPIError(exc.code, body, exc.headers) from exc

    return _request_with_retries(send, attempts=attempts)


def _extract_image_items(response: Dict[str, Any], auth: APIConfig, *, timeout: int, attempts: int) -> List[Dict[str, Any]]:
    data = response.get("data")
    if not isinstance(data, list) or not data:
        raise RuntimeError("Image API response did not contain data[].")

    images: List[Dict[str, Any]] = []
    for item in data:
        if not isinstance(item, dict):
            continue
        if item.get("b64_json"):
            images.append(
                {
                    "bytes": base64.b64decode(str(item["b64_json"])),
                    "revised_prompt": item.get("revised_prompt"),
                    "source": "b64_json",
                }
            )
            continue
        if item.get("url"):
            images.append(
                {
                    "bytes": _download_image_url(str(item["url"]), auth, timeout=timeout, attempts=attempts),
                    "revised_prompt": item.get("revised_prompt"),
                    "source": "url",
                }
            )

    if not images:
        raise RuntimeError("Image API response did not contain b64_json or url image data.")
    return images


def _mime_type_from_path(path: Path) -> str:
    lower = path.suffix.lower()
    if lower in {".jpg", ".jpeg"}:
        return "image/jpeg"
    if lower == ".webp":
        return "image/webp"
    if lower == ".gif":
        return "image/gif"
    if lower == ".png":
        return "image/png"
    guessed, _ = mimetypes.guess_type(str(path))
    return guessed or "application/octet-stream"


def _aionui_image_record(path: Path, *, result_bytes: int, workspace: Path, revised_prompt: Any = None) -> Dict[str, Any]:
    abs_path = path.resolve()
    record: Dict[str, Any] = {
        "saved_path": str(abs_path),
        "relative_path": _relative_to_workspace(abs_path, workspace),
        "image": {
            "path": str(abs_path),
            "mime_type": _mime_type_from_path(abs_path),
            "source": "codex_image_generation",
        },
        "result_omitted": True,
        "result_omitted_reason": "image_base64",
        "result_bytes": result_bytes,
    }
    if revised_prompt:
        record["revised_prompt"] = revised_prompt
    return record


def _write_image_items(
    image_items: List[Dict[str, Any]],
    outputs: List[Path],
    *,
    force: bool,
    downscale_max_dim: Optional[int],
    downscale_suffix: str,
    output_format: str,
    workspace: Path,
) -> List[Dict[str, Any]]:
    records: List[Dict[str, Any]] = []
    for idx, item in enumerate(image_items):
        if idx >= len(outputs):
            break
        out_path = outputs[idx]
        if out_path.exists() and not force:
            _die(f"Output already exists: {out_path} (use --force to overwrite)")
        out_path.parent.mkdir(parents=True, exist_ok=True)

        raw = item["bytes"]
        out_path.write_bytes(raw)
        print(f"Wrote {out_path}", file=sys.stderr)
        record = _aionui_image_record(
            out_path,
            result_bytes=len(raw),
            workspace=workspace,
            revised_prompt=item.get("revised_prompt"),
        )

        if downscale_max_dim is not None:
            derived = _derive_downscale_path(out_path, downscale_suffix)
            if derived.exists() and not force:
                _die(f"Output already exists: {derived} (use --force to overwrite)")
            derived.parent.mkdir(parents=True, exist_ok=True)
            resized = _downscale_image_bytes(raw, max_dim=downscale_max_dim, output_format=output_format)
            derived.write_bytes(resized)
            print(f"Wrote {derived}", file=sys.stderr)
            record["downscaled"] = _aionui_image_record(
                derived,
                result_bytes=len(resized),
                workspace=workspace,
                revised_prompt=item.get("revised_prompt"),
            )

        records.append(record)
    return records


def _state_file_path(args: argparse.Namespace, workspace: Path) -> Path:
    raw = args.state_file or DEFAULT_STATE_FILE
    return _resolve_path(raw, workspace)


def _read_state(path: Path) -> Dict[str, Any]:
    if not path.exists():
        return {"version": 1, "history": []}
    try:
        data = json.loads(path.read_text(encoding="utf-8"))
    except Exception:
        _warn(f"Could not parse state file; starting fresh: {path}")
        return {"version": 1, "history": []}
    return data if isinstance(data, dict) else {"version": 1, "history": []}


def _state_entry(
    record: Dict[str, Any],
    *,
    operation: str,
    prompt: str,
    response: Dict[str, Any],
    input_images: Optional[List[Path]] = None,
) -> Dict[str, Any]:
    entry = {
        "path": record["saved_path"],
        "relative_path": record.get("relative_path"),
        "operation": operation,
        "prompt": prompt,
        "revised_prompt": record.get("revised_prompt"),
        "requested_model": FIXED_MODEL,
        "model": response.get("model") or FIXED_MODEL,
        "created": response.get("created"),
        "output_format": response.get("output_format"),
        "size": response.get("size"),
        "quality": response.get("quality"),
        "usage": response.get("usage"),
    }
    if input_images:
        entry["input_images"] = [str(p.resolve()) for p in input_images]
    return {k: v for k, v in entry.items() if v is not None}


def _write_state(
    path: Path,
    records: List[Dict[str, Any]],
    *,
    operation: str,
    prompt: str,
    response: Dict[str, Any],
    input_images: Optional[List[Path]] = None,
) -> None:
    if not records:
        return
    state = _read_state(path)
    entries = [
        _state_entry(record, operation=operation, prompt=prompt, response=response, input_images=input_images)
        for record in records
    ]
    history = state.get("history")
    if not isinstance(history, list):
        history = []
    history.extend(entries)
    state["version"] = 1
    state["latest"] = entries[0]
    state["latest_outputs"] = entries
    state["history"] = history[-MAX_STATE_HISTORY:]
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(state, ensure_ascii=False, indent=2), encoding="utf-8")


def _resolve_latest_image(state_path: Path) -> Path:
    state = _read_state(state_path)
    latest = state.get("latest")
    path = latest.get("path") if isinstance(latest, dict) else None
    if not path:
        _die(f"No latest image found in state file: {state_path}")
    latest_path = Path(path)
    if not latest_path.exists():
        _die(f"Latest image from state file does not exist: {latest_path}")
    return latest_path.resolve()


def _check_image_paths(paths: Iterable[Path]) -> List[Path]:
    resolved: List[Path] = []
    for path in paths:
        if not path.exists():
            _die(f"Image file not found: {path}")
        if path.stat().st_size > MAX_IMAGE_BYTES:
            _warn(f"Image exceeds 50MB limit: {path}")
        resolved.append(path.resolve())
    return resolved


def _resolve_edit_image_paths(args: argparse.Namespace, workspace: Path, state_path: Path) -> List[Path]:
    images: List[Path] = []
    raw_images = args.image or []

    if args.from_last:
        images.append(_resolve_latest_image(state_path))

    for raw in raw_images:
        if raw.strip().lower() == "last":
            images.append(_resolve_latest_image(state_path))
        else:
            images.append(_resolve_path(raw, workspace))

    if not images:
        _die("Edit requires --image <path>, --image last, or --from-last.")
    return _check_image_paths(images)


def _build_common_payload(args: argparse.Namespace, prompt: str, *, include_input_fidelity: bool = False) -> Dict[str, Any]:
    payload: Dict[str, Any] = {
        "model": FIXED_MODEL,
        "prompt": prompt,
        "n": args.n,
        "size": args.size,
        "quality": args.quality,
        "background": args.background,
        "output_format": args.output_format,
        "output_compression": args.output_compression,
        "moderation": args.moderation,
        "response_format": args.response_format,
    }
    if include_input_fidelity:
        payload["input_fidelity"] = args.input_fidelity
    return {k: v for k, v in payload.items() if v is not None}


def _config_summary(auth: APIConfig) -> Dict[str, Any]:
    return {
        "base_url": auth.base_url,
        "base_url_source": auth.base_url_source,
        "api_key_source": auth.api_key_source,
        "provider": auth.provider,
    }


def _print_json(payload: Dict[str, Any]) -> None:
    print(json.dumps(payload, ensure_ascii=False, indent=2, sort_keys=True))


def _dry_run_result(
    *,
    operation: str,
    endpoint: str,
    auth: APIConfig,
    payload: Dict[str, Any],
    outputs: List[Path],
    workspace: Path,
    extra: Optional[Dict[str, Any]] = None,
) -> Dict[str, Any]:
    result: Dict[str, Any] = {
        "ok": True,
        "dry_run": True,
        "operation": operation,
        "endpoint": endpoint,
        "url": auth.base_url.rstrip("/") + endpoint,
        "config": _config_summary(auth),
        "payload": payload,
        "outputs": [_relative_to_workspace(path, workspace) for path in outputs],
    }
    if extra:
        result.update(extra)
    return result


def _final_result(
    *,
    operation: str,
    auth: APIConfig,
    response: Dict[str, Any],
    records: List[Dict[str, Any]],
    state_path: Path,
) -> Dict[str, Any]:
    result: Dict[str, Any] = {
        "ok": True,
        "operation": operation,
        "config": _config_summary(auth),
        "requested_model": FIXED_MODEL,
        "model": response.get("model") or FIXED_MODEL,
        "created": response.get("created"),
        "outputs": records,
        "latest": records[0]["saved_path"] if records else None,
        "state_file": str(state_path.resolve()),
        "usage": response.get("usage"),
    }
    if records:
        result["saved_path"] = records[0]["saved_path"]
        result["image"] = records[0]["image"]
        result["result_omitted"] = True
        result["result_omitted_reason"] = "image_base64"
        result["result_bytes"] = records[0]["result_bytes"]
    return {k: v for k, v in result.items() if v is not None}


def _attempts_for(args: argparse.Namespace) -> int:
    return int(getattr(args, "max_attempts", None) or args.http_retries)


def _generate(args: argparse.Namespace) -> None:
    prompt = _augment_prompt(args, _read_prompt(args.prompt, args.prompt_file))
    output_format = _normalize_output_format(args.output_format)
    args.output_format = output_format
    payload = _build_common_payload(args, prompt)
    _validate_payload_options(payload)

    workspace = args.workspace_path
    auth = args.api_config
    output_paths = _build_output_paths(
        out=args.out,
        output_format=output_format,
        count=args.n,
        out_dir=args.out_dir,
        prompt=prompt,
        workspace=workspace,
    )
    state_path = _state_file_path(args, workspace)

    if args.dry_run:
        _print_json(
            _dry_run_result(
                operation="generate",
                endpoint="/images/generations",
                auth=auth,
                payload=payload,
                outputs=output_paths,
                workspace=workspace,
            )
        )
        return

    print("Calling new-api Image API (generation). This can take up to a couple of minutes.", file=sys.stderr)
    started = time.time()
    response = _post_json(
        auth,
        "/images/generations",
        payload,
        timeout=args.timeout,
        attempts=_attempts_for(args),
    )
    elapsed = time.time() - started
    print(f"Generation completed in {elapsed:.1f}s.", file=sys.stderr)

    image_items = _extract_image_items(response, auth, timeout=args.timeout, attempts=_attempts_for(args))
    records = _write_image_items(
        image_items,
        output_paths,
        force=args.force,
        downscale_max_dim=args.downscale_max_dim,
        downscale_suffix=args.downscale_suffix,
        output_format=output_format,
        workspace=workspace,
    )
    _write_state(state_path, records, operation="generate", prompt=prompt, response=response)
    _print_json(_final_result(operation="generate", auth=auth, response=response, records=records, state_path=state_path))


def _edit(args: argparse.Namespace) -> None:
    prompt = _augment_prompt(args, _read_prompt(args.prompt, args.prompt_file))
    output_format = _normalize_output_format(args.output_format)
    args.output_format = output_format
    _validate_input_fidelity(args.input_fidelity)
    payload = _build_common_payload(args, prompt, include_input_fidelity=True)
    _validate_payload_options(payload, input_fidelity=args.input_fidelity)

    workspace = args.workspace_path
    auth = args.api_config
    state_path = _state_file_path(args, workspace)
    image_paths = _resolve_edit_image_paths(args, workspace, state_path)
    mask_path = _resolve_path(args.mask, workspace) if args.mask else None
    if mask_path:
        if not mask_path.exists():
            _die(f"Mask file not found: {mask_path}")
        if mask_path.suffix.lower() != ".png":
            _warn(f"Mask should be a PNG with an alpha channel: {mask_path}")
        if mask_path.stat().st_size > MAX_IMAGE_BYTES:
            _warn(f"Mask exceeds 50MB limit: {mask_path}")

    output_paths = _build_output_paths(
        out=args.out,
        output_format=output_format,
        count=args.n,
        out_dir=args.out_dir,
        prompt=prompt,
        workspace=workspace,
    )

    if args.dry_run:
        _print_json(
            _dry_run_result(
                operation="edit",
                endpoint="/images/edits",
                auth=auth,
                payload=payload,
                outputs=output_paths,
                workspace=workspace,
                extra={
                    "images": [str(p) for p in image_paths],
                    "mask": str(mask_path) if mask_path else None,
                },
            )
        )
        return

    print(f"Calling new-api Image API (edit) with {len(image_paths)} image(s).", file=sys.stderr)
    started = time.time()
    file_fields = [("image", image_paths[0])] if len(image_paths) == 1 else [("image[]", p) for p in image_paths]
    if mask_path is not None:
        file_fields.append(("mask", mask_path))
    response = _post_multipart(
        auth,
        "/images/edits",
        payload,
        file_fields,
        timeout=args.timeout,
        attempts=_attempts_for(args),
    )
    elapsed = time.time() - started
    print(f"Edit completed in {elapsed:.1f}s.", file=sys.stderr)

    image_items = _extract_image_items(response, auth, timeout=args.timeout, attempts=_attempts_for(args))
    records = _write_image_items(
        image_items,
        output_paths,
        force=args.force,
        downscale_max_dim=args.downscale_max_dim,
        downscale_suffix=args.downscale_suffix,
        output_format=output_format,
        workspace=workspace,
    )
    _write_state(state_path, records, operation="edit", prompt=prompt, response=response, input_images=image_paths)
    _print_json(_final_result(operation="edit", auth=auth, response=response, records=records, state_path=state_path))


async def _run_generate_batch(args: argparse.Namespace) -> int:
    jobs = _read_jobs_jsonl(args.input)
    workspace = args.workspace_path
    auth = args.api_config
    out_dir = _resolve_path(args.out_dir, workspace)
    state_path = _state_file_path(args, workspace)

    base_fields = _fields_from_args(args)
    base_payload = {
        "model": FIXED_MODEL,
        "n": args.n,
        "size": args.size,
        "quality": args.quality,
        "background": args.background,
        "output_format": args.output_format,
        "output_compression": args.output_compression,
        "moderation": args.moderation,
        "response_format": args.response_format,
    }

    if args.dry_run:
        dry_jobs = []
        for i, job in enumerate(jobs, start=1):
            prompt = str(job["prompt"]).strip()
            fields = _merge_non_null(base_fields, job.get("fields", {}))
            fields = _merge_non_null(fields, {k: job.get(k) for k in base_fields.keys()})
            augmented = _augment_prompt_fields(args.augment, prompt, fields)

            job_payload = dict(base_payload)
            job_payload["prompt"] = augmented
            job_payload = _merge_non_null(job_payload, {k: job.get(k) for k in base_payload.keys()})
            job_payload = {k: v for k, v in job_payload.items() if v is not None}

            effective_output_format = _normalize_output_format(job_payload.get("output_format"))
            job_payload["output_format"] = effective_output_format
            _validate_payload_options(job_payload)

            n = int(job_payload.get("n", 1))
            outputs = _job_output_paths(
                out_dir=out_dir,
                output_format=effective_output_format,
                idx=i,
                prompt=prompt,
                n=n,
                explicit_out=job.get("out"),
            )
            dry_jobs.append(
                {
                    "job": i,
                    "endpoint": "/images/generations",
                    "payload": job_payload,
                    "outputs": [_relative_to_workspace(p, workspace) for p in outputs],
                }
            )
        _print_json({"ok": True, "dry_run": True, "operation": "generate-batch", "config": _config_summary(auth), "jobs": dry_jobs})
        return 0

    sem = asyncio.Semaphore(args.concurrency)
    any_failed = False

    async def run_job(i: int, job: Dict[str, Any]) -> Dict[str, Any]:
        nonlocal any_failed
        prompt = str(job["prompt"]).strip()
        job_label = f"[job {i}/{len(jobs)}]"

        fields = _merge_non_null(base_fields, job.get("fields", {}))
        fields = _merge_non_null(fields, {k: job.get(k) for k in base_fields.keys()})
        augmented = _augment_prompt_fields(args.augment, prompt, fields)

        payload = dict(base_payload)
        payload["prompt"] = augmented
        payload = _merge_non_null(payload, {k: job.get(k) for k in base_payload.keys()})
        payload = {k: v for k, v in payload.items() if v is not None}

        effective_output_format = _normalize_output_format(payload.get("output_format"))
        payload["output_format"] = effective_output_format
        _validate_payload_options(payload)
        n = int(payload.get("n", 1))
        outputs = _job_output_paths(
            out_dir=out_dir,
            output_format=effective_output_format,
            idx=i,
            prompt=prompt,
            n=n,
            explicit_out=job.get("out"),
        )

        try:
            async with sem:
                print(f"{job_label} starting", file=sys.stderr)
                started = time.time()
                response = await asyncio.to_thread(
                    _post_json,
                    auth,
                    "/images/generations",
                    payload,
                    timeout=args.timeout,
                    attempts=_attempts_for(args),
                )
                elapsed = time.time() - started
                print(f"{job_label} completed in {elapsed:.1f}s", file=sys.stderr)
            image_items = _extract_image_items(response, auth, timeout=args.timeout, attempts=_attempts_for(args))
            records = _write_image_items(
                image_items,
                outputs,
                force=args.force,
                downscale_max_dim=args.downscale_max_dim,
                downscale_suffix=args.downscale_suffix,
                output_format=effective_output_format,
                workspace=workspace,
            )
            return {"job": i, "ok": True, "response": response, "outputs": records, "prompt": augmented}
        except Exception as exc:
            any_failed = True
            print(f"{job_label} failed: {exc}", file=sys.stderr)
            if args.fail_fast:
                raise
            return {"job": i, "ok": False, "error": str(exc)}

    tasks = [asyncio.create_task(run_job(i, job)) for i, job in enumerate(jobs, start=1)]
    try:
        results = await asyncio.gather(*tasks)
    except Exception:
        for task in tasks:
            if not task.done():
                task.cancel()
        raise

    successful_records: List[Dict[str, Any]] = []
    state_response: Dict[str, Any] = {}
    state_prompt = ""
    for result in results:
        if not result.get("ok"):
            continue
        records = result.get("outputs") or []
        if records and not successful_records:
            state_response = result.get("response") or {}
            state_prompt = result.get("prompt") or ""
        successful_records.extend(records)
    if successful_records:
        _write_state(state_path, successful_records, operation="generate-batch", prompt=state_prompt, response=state_response)

    _print_json(
        {
            "ok": not any_failed,
            "operation": "generate-batch",
            "config": _config_summary(auth),
            "requested_model": FIXED_MODEL,
            "jobs": results,
            "state_file": str(state_path.resolve()),
        }
    )
    return 1 if any_failed else 0


def _generate_batch(args: argparse.Namespace) -> None:
    exit_code = asyncio.run(_run_generate_batch(args))
    if exit_code:
        raise SystemExit(exit_code)


def _add_shared_args(parser: argparse.ArgumentParser) -> None:
    parser.add_argument("--model", dest="legacy_model", help=argparse.SUPPRESS)
    parser.add_argument("--base-url")
    parser.add_argument("--api-key")
    parser.add_argument("--provider")
    parser.add_argument("--workspace")
    parser.add_argument("--state-file")
    parser.add_argument("--timeout", type=int, default=300)
    parser.add_argument("--http-retries", type=int, default=3)
    parser.add_argument("--print-json", action="store_true", help=argparse.SUPPRESS)
    parser.add_argument("--prompt")
    parser.add_argument("--prompt-file")
    parser.add_argument("--n", type=int, default=1)
    parser.add_argument("--size", default=DEFAULT_SIZE)
    parser.add_argument("--quality", default=DEFAULT_QUALITY)
    parser.add_argument("--background")
    parser.add_argument("--output-format", default=DEFAULT_OUTPUT_FORMAT)
    parser.add_argument("--output-compression", type=int)
    parser.add_argument("--response-format", default=DEFAULT_RESPONSE_FORMAT)
    parser.add_argument("--moderation")
    parser.add_argument("--out")
    parser.add_argument("--out-dir")
    parser.add_argument("--force", action="store_true")
    parser.add_argument("--dry-run", action="store_true")
    parser.add_argument("--augment", dest="augment", action="store_true")
    parser.add_argument("--no-augment", dest="augment", action="store_false")
    parser.set_defaults(augment=True)

    parser.add_argument("--use-case")
    parser.add_argument("--scene")
    parser.add_argument("--subject")
    parser.add_argument("--style")
    parser.add_argument("--composition")
    parser.add_argument("--lighting")
    parser.add_argument("--palette")
    parser.add_argument("--materials")
    parser.add_argument("--text")
    parser.add_argument("--constraints")
    parser.add_argument("--negative")

    parser.add_argument("--downscale-max-dim", type=int)
    parser.add_argument("--downscale-suffix", default=DEFAULT_DOWNSCALE_SUFFIX)


def _validate_args(args: argparse.Namespace) -> None:
    _validate_legacy_model_arg(args)
    if args.n < 1 or args.n > 10:
        _die("--n must be between 1 and 10")
    if getattr(args, "concurrency", 1) < 1 or getattr(args, "concurrency", 1) > 25:
        _die("--concurrency must be between 1 and 25")
    if getattr(args, "max_attempts", 3) is not None and (
        getattr(args, "max_attempts", 3) < 1 or getattr(args, "max_attempts", 3) > 10
    ):
        _die("--max-attempts must be between 1 and 10")
    if args.http_retries < 1 or args.http_retries > 10:
        _die("--http-retries must be between 1 and 10")
    if args.timeout < 1:
        _die("--timeout must be >= 1")
    if args.output_compression is not None and not (0 <= args.output_compression <= 100):
        _die("--output-compression must be between 0 and 100")
    if args.command == "generate-batch" and not args.out_dir:
        _die("generate-batch requires --out-dir")
    if getattr(args, "downscale_max_dim", None) is not None and args.downscale_max_dim < 1:
        _die("--downscale-max-dim must be >= 1")
    _validate_gpt_image_2_size(args.size)
    _validate_quality(args.quality)
    _validate_background(args.background)
    _validate_response_format(args.response_format)
    _validate_model_specific_options(
        background=args.background,
        input_fidelity=getattr(args, "input_fidelity", None),
    )


def main() -> int:
    parser = argparse.ArgumentParser(
        description="Generate or edit images through new-api / OpenAI-compatible Images API"
    )
    subparsers = parser.add_subparsers(dest="command", required=True)

    gen_parser = subparsers.add_parser("generate", help="Create a new image")
    _add_shared_args(gen_parser)
    gen_parser.set_defaults(func=_generate)

    batch_parser = subparsers.add_parser("generate-batch", help="Generate multiple prompts concurrently from JSONL")
    _add_shared_args(batch_parser)
    batch_parser.add_argument("--input", required=True, help="Path to JSONL file (one job per line)")
    batch_parser.add_argument("--concurrency", type=int, default=DEFAULT_CONCURRENCY)
    batch_parser.add_argument("--max-attempts", type=int, default=3)
    batch_parser.add_argument("--fail-fast", action="store_true")
    batch_parser.set_defaults(func=_generate_batch)

    edit_parser = subparsers.add_parser("edit", help="Edit an existing image")
    _add_shared_args(edit_parser)
    edit_parser.add_argument("--image", action="append")
    edit_parser.add_argument("--from-last", action="store_true")
    edit_parser.add_argument("--mask")
    edit_parser.add_argument("--input-fidelity")
    edit_parser.set_defaults(func=_edit)

    args = parser.parse_args()
    _validate_args(args)
    args.workspace_path = _resolve_workspace(args)
    args.api_config = _resolve_api_config(args, args.workspace_path)
    args.func(args)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
