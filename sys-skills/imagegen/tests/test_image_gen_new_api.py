from __future__ import annotations

import contextlib
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import importlib.util
import json
from pathlib import Path
import subprocess
import sys
import tempfile
import threading
import unittest


SKILL_ROOT = Path(__file__).resolve().parents[1]
SCRIPT = SKILL_ROOT / "scripts" / "image_gen.py"
PNG_B64 = (
    "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8"
    "/5+hHgAHggJ/PchI7wAAAABJRU5ErkJggg=="
)


def load_module():
    spec = importlib.util.spec_from_file_location("image_gen", SCRIPT)
    module = importlib.util.module_from_spec(spec)
    assert spec and spec.loader
    sys.modules["image_gen"] = module
    spec.loader.exec_module(module)
    return module


class ImageHandler(BaseHTTPRequestHandler):
    def do_POST(self):
        length = int(self.headers.get("Content-Length", "0"))
        body = self.rfile.read(length)
        self.server.requests.append(  # type: ignore[attr-defined]
            {
                "path": self.path,
                "headers": dict(self.headers),
                "body": body,
            }
        )

        if self.path == "/v1/images/generations":
            payload = json.loads(body.decode("utf-8"))
            if payload.get("model") != "gpt-image-2":
                self.send_error(400, "bad model")
                return
            self._send_image_response(payload.get("prompt", ""))
            return

        if self.path == "/v1/images/edits":
            if b'name="model"' not in body or b"gpt-image-2" not in body:
                self.send_error(400, "bad model")
                return
            if b'name="image"' not in body and b'name="image[]"' not in body:
                self.send_error(400, "missing image")
                return
            self._send_image_response("edited")
            return

        self.send_error(404)

    def _send_image_response(self, revised_prompt: str):
        response = {
            "created": 1789370877,
            "data": [{"b64_json": PNG_B64, "revised_prompt": revised_prompt}],
            "background": "auto",
            "output_format": "png",
            "quality": "auto",
            "size": "1024x1024",
            "model": "gpt-image-2-codex",
            "usage": {"input_tokens": 1, "output_tokens": 2, "total_tokens": 3},
        }
        raw = json.dumps(response).encode("utf-8")
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(raw)))
        self.end_headers()
        self.wfile.write(raw)

    def log_message(self, fmt, *args):
        return


@contextlib.contextmanager
def image_server():
    server = ThreadingHTTPServer(("127.0.0.1", 0), ImageHandler)
    server.requests = []  # type: ignore[attr-defined]
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    try:
        yield server, f"http://127.0.0.1:{server.server_port}"
    finally:
        server.shutdown()
        server.server_close()
        thread.join(timeout=5)


def run_cli(cwd: Path, args: list[str]) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        [sys.executable, str(SCRIPT), *args],
        cwd=cwd,
        text=True,
        encoding="utf-8",
        capture_output=True,
        check=False,
    )


class ImageGenNewAPITest(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.mod = load_module()

    def test_normalize_base_url(self):
        cases = {
            "https://hth.huaqing.run": "https://hth.huaqing.run/v1",
            "https://hth.huaqing.run/v1/": "https://hth.huaqing.run/v1",
            "https://hth.huaqing.run/v1/images/generations": "https://hth.huaqing.run/v1",
            "https://hth.huaqing.run/root": "https://hth.huaqing.run/root/v1",
        }
        for raw, expected in cases.items():
            with self.subTest(raw=raw):
                self.assertEqual(self.mod._normalize_base_url(raw), expected)

    def test_opencode_jsonc_config_parsing(self):
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "opencode.jsonc"
            path.write_text(
                """
                {
                  // URL comments must not break parsing.
                  "model": "hth/gpt-5.6-terra",
                  "provider": {
                    "hth": {
                      "api": "https://hth.huaqing.run/v1",
                      "options": {
                        "apiKey": "sk-test",
                      },
                    },
                  },
                }
                """,
                encoding="utf-8",
            )
            cfg = self.mod._read_opencode_config(path)
            self.assertEqual(cfg.provider, "hth")
            self.assertEqual(cfg.base_url, "https://hth.huaqing.run/v1")
            self.assertEqual(cfg.api_key, "sk-test")

    def test_rejects_non_fixed_model(self):
        with tempfile.TemporaryDirectory() as tmp:
            result = run_cli(
                Path(tmp),
                [
                    "generate",
                    "--model",
                    "gpt-image-1",
                    "--prompt",
                    "a cat",
                    "--size",
                    "1024x1024",
                    "--dry-run",
                ],
            )
            self.assertNotEqual(result.returncode, 0)
            self.assertIn("model is fixed to gpt-image-2", result.stderr)

    def test_generate_writes_aionui_output_and_state(self):
        with tempfile.TemporaryDirectory() as tmp, image_server() as (server, base_url):
            workspace = Path(tmp)
            result = run_cli(
                workspace,
                [
                    "generate",
                    "--base-url",
                    base_url,
                    "--api-key",
                    "sk-secret",
                    "--prompt",
                    "a cat",
                    "--size",
                    "1024x1024",
                    "--out",
                    "output/imagegen/cat.png",
                    "--no-augment",
                    "--force",
                ],
            )
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertNotIn("sk-secret", result.stdout + result.stderr)

            request = server.requests[0]  # type: ignore[attr-defined]
            self.assertEqual(request["path"], "/v1/images/generations")
            payload = json.loads(request["body"].decode("utf-8"))
            self.assertEqual(payload["model"], "gpt-image-2")
            self.assertEqual(payload["prompt"], "a cat")

            out = workspace / "output" / "imagegen" / "cat.png"
            self.assertTrue(out.exists())
            self.assertTrue(out.read_bytes().startswith(b"\x89PNG"))

            parsed = json.loads(result.stdout)
            self.assertEqual(parsed["requested_model"], "gpt-image-2")
            self.assertEqual(parsed["model"], "gpt-image-2-codex")
            self.assertEqual(parsed["image"]["mime_type"], "image/png")
            self.assertEqual(parsed["image"]["source"], "codex_image_generation")
            self.assertTrue(parsed["result_omitted"])

            state = json.loads((workspace / "output" / "imagegen" / ".imagegen-state.json").read_text(encoding="utf-8"))
            self.assertEqual(state["latest"]["path"], str(out.resolve()))
            self.assertNotIn("sk-secret", json.dumps(state))

    def test_edit_from_last_sends_multipart_and_updates_latest(self):
        with tempfile.TemporaryDirectory() as tmp, image_server() as (server, base_url):
            workspace = Path(tmp)
            first = run_cli(
                workspace,
                [
                    "generate",
                    "--base-url",
                    base_url,
                    "--api-key",
                    "sk-secret",
                    "--prompt",
                    "a cat",
                    "--size",
                    "1024x1024",
                    "--out",
                    "output/imagegen/cat.png",
                    "--no-augment",
                    "--force",
                ],
            )
            self.assertEqual(first.returncode, 0, first.stderr)

            edit = run_cli(
                workspace,
                [
                    "edit",
                    "--base-url",
                    base_url,
                    "--api-key",
                    "sk-secret",
                    "--from-last",
                    "--prompt",
                    "make it watercolor",
                    "--size",
                    "1024x1024",
                    "--out",
                    "output/imagegen/cat-edit.png",
                    "--no-augment",
                    "--force",
                ],
            )
            self.assertEqual(edit.returncode, 0, edit.stderr)

            request = server.requests[-1]  # type: ignore[attr-defined]
            self.assertEqual(request["path"], "/v1/images/edits")
            content_type = request["headers"]["Content-Type"]
            self.assertIn("multipart/form-data", content_type)
            body = request["body"]
            self.assertIn(b'name="model"', body)
            self.assertIn(b"gpt-image-2", body)
            self.assertIn(b'name="image"; filename="cat.png"', body)

            parsed = json.loads(edit.stdout)
            self.assertTrue(parsed["saved_path"].endswith("cat-edit.png"))
            state = json.loads((workspace / "output" / "imagegen" / ".imagegen-state.json").read_text(encoding="utf-8"))
            self.assertTrue(state["latest"]["path"].endswith("cat-edit.png"))


if __name__ == "__main__":
    unittest.main()
