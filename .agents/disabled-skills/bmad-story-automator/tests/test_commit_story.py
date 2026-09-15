from __future__ import annotations

import tempfile
import unittest
from pathlib import Path

from story_automator.commands.basic import _paths_from_story_file, _resolve_story_file, _story_commit_paths


def _write(path: Path, content: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(content, encoding="utf-8")


class CommitStoryTests(unittest.TestCase):
    def setUp(self) -> None:
        self.tmp = tempfile.TemporaryDirectory()
        self.root = Path(self.tmp.name)
        story = self.root / "_bmad-output" / "implementation-artifacts" / "7B-1-sample-story.md"
        _write(
            story,
            """---
Status: done
---

### File List

- `_bmad-output/implementation-artifacts/7B-1-sample-story.md`
- `service/example.go`
- `web/default/src/example.tsx`

## Change Log
""",
        )

    def tearDown(self) -> None:
        self.tmp.cleanup()

    def test_resolve_story_file_supports_dotted_story_id(self) -> None:
        self.assertEqual(
            _resolve_story_file(str(self.root), "7B.1"),
            "_bmad-output/implementation-artifacts/7B-1-sample-story.md",
        )

    def test_paths_from_story_file_reads_file_list_section(self) -> None:
        self.assertEqual(
            _paths_from_story_file(str(self.root), "_bmad-output/implementation-artifacts/7B-1-sample-story.md"),
            [
                "_bmad-output/implementation-artifacts/7B-1-sample-story.md",
                "service/example.go",
                "web/default/src/example.tsx",
            ],
        )

    def test_story_commit_paths_includes_story_file_once(self) -> None:
        self.assertEqual(
            _story_commit_paths(str(self.root), "7B.1"),
            [
                "_bmad-output/implementation-artifacts/7B-1-sample-story.md",
                "service/example.go",
                "web/default/src/example.tsx",
            ],
        )


if __name__ == "__main__":
    unittest.main()
