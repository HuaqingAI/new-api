#!/usr/bin/env python
"""Cherry Studio knowledge base API wrapper."""

from __future__ import annotations

import argparse
import json
import sys
import urllib.error
import urllib.parse
import urllib.request
from pathlib import Path
from typing import Any


SKILL_DIR = Path(__file__).resolve().parents[1]
CONFIG_PATH = SKILL_DIR / "config.json"


def load_config() -> dict[str, Any]:
    if not CONFIG_PATH.exists():
        raise SystemExit(f"Config file not found: {CONFIG_PATH}")
    with CONFIG_PATH.open("r", encoding="utf-8") as f:
        return json.load(f)


def build_headers(config: dict[str, Any]) -> dict[str, str]:
    headers = {"Accept": "application/json"}
    auth = config.get("auth", {})
    api_key = auth.get("api_key")
    if api_key:
        header_name = auth.get("authorization_header") or "Authorization"
        scheme = auth.get("authorization_scheme") or "Bearer"
        headers[header_name] = f"{scheme} {api_key}" if scheme else api_key
    return headers


def request_json(
    config: dict[str, Any],
    method: str,
    path: str,
    *,
    query: dict[str, Any] | None = None,
    body: dict[str, Any] | None = None,
) -> Any:
    api = config.get("api", {})
    base_url = (api.get("base_url") or "http://127.0.0.1:23333").rstrip("/")
    timeout = float(api.get("timeout_seconds") or 60)
    url = f"{base_url}{path}"
    if query:
        url = f"{url}?{urllib.parse.urlencode(query)}"

    data = None
    headers = build_headers(config)
    if body is not None:
        data = json.dumps(body, ensure_ascii=False).encode("utf-8")
        headers["Content-Type"] = "application/json"

    req = urllib.request.Request(url, data=data, headers=headers, method=method)
    try:
        with urllib.request.urlopen(req, timeout=timeout) as response:
            raw = response.read().decode("utf-8")
    except urllib.error.HTTPError as exc:
        error_body = exc.read().decode("utf-8", errors="replace")
        raise SystemExit(f"HTTP {exc.code} {exc.reason}: {error_body}") from exc
    except urllib.error.URLError as exc:
        raise SystemExit(f"Request failed: {exc.reason}") from exc

    return json.loads(raw) if raw else None


def print_json(value: Any) -> None:
    print(json.dumps(value, ensure_ascii=False, indent=2))


def list_bases(args: argparse.Namespace) -> None:
    config = load_config()
    defaults = config.get("defaults", {})
    query = {
        "limit": args.limit if args.limit is not None else defaults.get("list_limit", 100),
        "offset": args.offset if args.offset is not None else defaults.get("list_offset", 0),
    }
    print_json(request_json(config, "GET", "/v1/knowledge-bases", query=query))


def get_base(args: argparse.Namespace) -> None:
    config = load_config()
    kb_id = urllib.parse.quote(args.knowledge_base_id, safe="")
    print_json(request_json(config, "GET", f"/v1/knowledge-bases/{kb_id}"))


def search(args: argparse.Namespace) -> None:
    config = load_config()
    defaults = config.get("defaults", {})
    knowledge_base_ids = args.knowledge_base_id
    if not knowledge_base_ids:
        knowledge_base_ids = config.get("knowledge_base_ids") or []

    body: dict[str, Any] = {
        "query": args.query,
        "document_count": args.count if args.count is not None else defaults.get("document_count", 5),
    }
    if knowledge_base_ids:
        body["knowledge_base_ids"] = knowledge_base_ids

    print_json(request_json(config, "POST", "/v1/knowledge-bases/search", body=body))


def main(argv: list[str]) -> int:
    parser = argparse.ArgumentParser(description="Search Cherry Studio knowledge bases")
    subparsers = parser.add_subparsers(dest="command", required=True)

    list_parser = subparsers.add_parser("list", help="List knowledge bases")
    list_parser.add_argument("--limit", type=int)
    list_parser.add_argument("--offset", type=int)
    list_parser.set_defaults(func=list_bases)

    get_parser = subparsers.add_parser("get", help="Get a knowledge base by ID")
    get_parser.add_argument("knowledge_base_id")
    get_parser.set_defaults(func=get_base)

    search_parser = subparsers.add_parser("search", help="Search knowledge bases")
    search_parser.add_argument("query")
    search_parser.add_argument("--count", type=int, help="Maximum number of chunks to return")
    search_parser.add_argument(
        "--knowledge-base-id",
        action="append",
        default=[],
        help="Knowledge base ID to search. Repeat to search multiple bases.",
    )
    search_parser.set_defaults(func=search)

    args = parser.parse_args(argv)
    args.func(args)
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
