from __future__ import annotations

import json
import os
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

from story_automator.commands.orchestrator import _normalize_key, _story_file_status, _verify_step
from story_automator.core.review_verify import verify_code_review_completion
from story_automator.core.story_keys import normalize_story_key


STATE_FRONTMATTER = """---
epic: "1"
epicName: "Agent Platform - Epic Breakdown"
epicSource: "{epic_source}"
storyRange: ["1.1", "1.2"]
status: "IN_PROGRESS"
---
"""

SPRINT_STATUS = """development_status:
  epic-1: done
  1-1-view-enterprise-department-tree: done
  1-2-maintain-user-department-memberships: done

  agent-platform-epic-1: done
  ap-1-1-establish-shared-resource-registry-and-stable-identity: done
  ap-1-2-implement-typed-detail-storage-for-skill-knowledge-agent: done

  epic-7b: in-progress
  7b-1-decouple-governance-action-result-from-notification-delivery: done
  7b-2-redesign-budget-pool-selection-and-selected-summary: backlog
"""


def _write(path: Path, content: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(content, encoding="utf-8")


class ContextualStoryKeyTests(unittest.TestCase):
    def setUp(self) -> None:
        self.tmp = tempfile.TemporaryDirectory()
        self.root = Path(self.tmp.name)
        self.impl = self.root / "_bmad-output" / "implementation-artifacts"
        self.state_file = self.root / "_bmad-output" / "story-automator" / "orchestration-1.md"
        self.old_pwd = Path(os.getcwd())

        _write(self.impl / "sprint-status.yaml", SPRINT_STATUS)
        _write(
            self.impl / "1-1-view-enterprise-department-tree.md",
            "---\nStatus: done\nTitle: Legacy 1.1\n---\n",
        )
        _write(
            self.impl / "ap-1-1-establish-shared-resource-registry-and-stable-identity.md",
            "---\nStatus: done\nTitle: Agent Platform 1.1\n---\n",
        )
        _write(
            self.impl / "ap-1-2-implement-typed-detail-storage-for-skill-knowledge-agent.md",
            "---\nStatus: done\nTitle: Agent Platform 1.2\n---\n",
        )
        _write(
            self.state_file,
            STATE_FRONTMATTER.format(
                epic_source=self.root / "_bmad-output" / "planning-artifacts" / "epics-agent-platform.md"
            ),
        )
        (self.root / "_bmad-output" / "planning-artifacts").mkdir(parents=True, exist_ok=True)
        (self.root / "_bmad-output" / "planning-artifacts" / "epics-agent-platform.md").write_text(
            "# agent platform\n",
            encoding="utf-8",
        )

    def tearDown(self) -> None:
        os.chdir(self.old_pwd)
        self.tmp.cleanup()

    def test_normalize_story_key_uses_state_context(self) -> None:
        result = normalize_story_key(str(self.root), "1.1", state_file=self.state_file)
        self.assertIsNotNone(result)
        assert result is not None
        self.assertEqual(result.key, "ap-1-1-establish-shared-resource-registry-and-stable-identity")
        self.assertEqual(result.prefix, "ap-1-1")

    def test_review_completion_uses_state_context(self) -> None:
        payload = verify_code_review_completion(str(self.root), "1.1", state_file=self.state_file)
        self.assertTrue(payload["verified"])
        self.assertEqual(payload["story"], "ap-1-1-establish-shared-resource-registry-and-stable-identity")

    def test_verify_step_create_uses_state_context(self) -> None:
        exit_code = _verify_step(["create", "1.1", "--state-file", str(self.state_file)])
        self.assertEqual(exit_code, 0)

    def test_normalize_key_helper_uses_state_context(self) -> None:
        os.chdir(self.root)
        captured: list[dict[str, object]] = []
        with patch("story_automator.commands.orchestrator.print_json", side_effect=captured.append):
            exit_code = _normalize_key(["1.1", "--state-file", str(self.state_file)])
        self.assertEqual(exit_code, 0)
        self.assertEqual(captured[-1]["key"], "ap-1-1-establish-shared-resource-registry-and-stable-identity")

    def test_story_file_status_uses_state_context(self) -> None:
        os.chdir(self.root)
        captured: list[dict[str, object]] = []
        with patch("story_automator.commands.orchestrator.print_json", side_effect=captured.append):
            exit_code = _story_file_status(["1.1", "--state-file", str(self.state_file)])
        self.assertEqual(exit_code, 0)
        self.assertEqual(captured[-1]["story_key"], "ap-1-1-establish-shared-resource-registry-and-stable-identity")
        self.assertTrue(str(captured[-1]["file"]).endswith("ap-1-1-establish-shared-resource-registry-and-stable-identity.md"))

    def test_normalize_story_key_supports_alpha_numeric_epic_ids(self) -> None:
        result = normalize_story_key(str(self.root), "7B.1")
        self.assertIsNotNone(result)
        assert result is not None
        self.assertEqual(result.id, "7B.1")
        self.assertEqual(result.prefix, "7b-1")
        self.assertEqual(result.key, "7b-1-decouple-governance-action-result-from-notification-delivery")


if __name__ == "__main__":
    unittest.main()
