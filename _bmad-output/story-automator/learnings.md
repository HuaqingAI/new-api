## Run: 2026-05-30T18:13:37Z

**Epic:** new-api - Epic Breakdown
**Stories:** 3.1, 3.2, 3.3, 3.4, 3.5, 4.1, 4.2, 4.3, 4.4, 4.5

### Patterns Observed
- Most stories completed cleanly once source-of-truth verification was used after each child session.
- The main orchestration risk came from session/runtime bookkeeping rather than unfinished product work.

### Code Review Insights
- Common issues: retrying stalled child sessions, keeping sprint-status aligned, and validating completion from source-of-truth files.
- Average cycles to clean: low overall, with 2 additional review cycles recorded across the run.

### Timing Estimates
- create-story: varied, but usually short once the child session started correctly.
- dev-story: moderate, with a few stories needing one retry before sprint status advanced.
- code-review: usually one pass, with occasional extra cycles when source-of-truth lagged behind session output.

### Recommendations for Future Runs
- Keep recovery checks on the shared sprint-status parser so resume logic and verification logic cannot drift.
- Auto-finalize orchestration state when all stories and retrospectives are already done to avoid stale `IN_PROGRESS` state files.
## Run: 2026-05-30T23:56:15Z

**Epic:** new-api - Epic Breakdown
**Stories:** 5.1, 5.2, 5.3, 5.4, 5.5, 5.6

### Patterns Observed
- Source-of-truth verification via sprint-status and story artifacts was more reliable than tmux monitor state during the back half of Epic 5.
- The most fragile orchestration steps were create-story and retrospective, not the actual dev or review implementation loops.

### Code Review Insights
- Common issues: create-story stalls, final status sync lagging behind implementation evidence, and shared time-window contract drift between usage and alerts.
- Average cycles to clean: low-to-moderate overall, with 4 review cycles recorded across stories 5.4-5.6.

### Timing Estimates
- create-story: usually short when healthy, but repeated stalls required manual takeover for 5.5 and 5.6.
- dev-story: generally steady once story artifacts existed and sprint status advanced correctly.
- code-review: typically one pass, with extra cycles when source-of-truth did not sync immediately or contract hardening fixes were needed.

### Recommendations for Future Runs
- Let sprint-status/story artifact verification outrank child-session liveness when deciding whether to retry, finalize, or wrap up a story.
- Add explicit self-healing for repeated create-story stalls so the automator can switch to manual artifact generation logic sooner.
- Keep dedicated regression coverage for shared `[from, to)` boundaries anywhere enterprise usage and alerts reuse the same facts.

## Run: 2026-06-03T02:30:00Z

**Epic:** Agent Platform - Epic Breakdown
**Stories:** 6.1, 6.2, 6.3, 6.4, 6.5, 6.6, 6.7, 6.8

### Patterns Observed
- Codex session execution was generally stable after the CODEX_HOME auth/config symlink fix; the main remaining friction was verifier drift around AP sprint keys that use `ap-*` rather than plain `6.*` aliases.
- Source-of-truth checks against story artifacts and sprint-status were essential to recover from monitor/workflow_not_complete cases without blocking the run.

### Code Review Insights
- Common issues: review verification mismatched normalized story aliases vs real sprint keys for AP-6.6 through AP-6.8, requiring source-of-truth confirmation.
- Average cycles to clean: ~1.5

### Timing Estimates
- create-story: ~6m
- dev-story: ~13m
- code-review: ~8m per cycle

### Recommendations for Future Runs
- Teach review/finalization helpers to resolve real AP sprint keys (`ap-*`) before declaring workflow_not_complete.
- Keep using direct sprint-status/story-artifact verification immediately after each step whenever monitor output is incomplete.

## Run: 2026-06-03T08:16:30Z

**Epic:** new-api - Epic Breakdown
**Stories:** 7B.1, 7B.2, 7B.3, 7B.4, 7B.5

### Patterns Observed
- Source-of-truth verification from `sprint-status.yaml` and story artifacts repeatedly outperformed child-session liveness as the signal for whether a story was actually complete.
- Frontend/i18n-heavy stories often finished the substantive work before codex child sessions self-terminated, so a second lightweight review pass was useful for final status synchronization.

### Code Review Insights
- Common issues: incomplete story-key normalization in the helper layer, long-lived idle review/create sessions, and UI/i18n regressions surfacing only in targeted locale tests.
- Average cycles to clean: low for implementation quality, but orchestration needed 3 extra review/create recovery cycles across the epic.

### Timing Estimates
- create-story: usually short once the story artifact actually started writing; stalls were mostly session lifecycle issues.
- dev-story: steady once the story artifact existed and sprint status moved to `in-progress`.
- code-review: usually one substantive pass, with occasional second passes only to complete status synchronization.

### Recommendations for Future Runs
- Keep the repaired story-automator helper changes for contextual story keys, alphanumeric epic IDs, and story-scoped commits.
- For frontend governance stories, continue prioritizing source verification after tests pass instead of waiting indefinitely for idle codex runners.
- Add explicit session cleanup/logging in the state doc to reduce confusion from stale tmux sessions on later resume attempts.

## Run: 2026-06-03T16:01:47Z

**Epic:** new-api - Epic Breakdown
**Stories:** 7B.6, 7B.7

### Patterns Observed
- Source-of-truth verification remained the strongest completion signal: 7B.7 dev advanced via `sprint-status=review`, and review completed via monitor `verified_complete` plus `sprint-status=done`.
- The substantive dev/automate/review loops completed cleanly, while parser sub-agent calls and retrospective timeout were the only orchestration weak points.

### Code Review Insights
- Common issues: role-gating edge cases, frontend visibility for non-admin budget users, and wording/i18n consistency for subordinate allocation copy.
- Average cycles to clean: one substantive review cycle for 7B.7 after automate guardrails were generated.

### Timing Estimates
- create-story: short once the story file session launched.
- dev-story: moderate; status moved correctly to review after implementation.
- code-review: one cycle, with Go/frontend/i18n validation and an auto-fix for non-admin budget overview layout.

### Recommendations for Future Runs
- Keep treating parse-output failure as non-blocking when monitor output and sprint-status agree.
- Consider a shorter, explicit retrospective background policy so active retro sessions can be logged without confusing the main orchestration state.
- Preserve the API/UI/i18n guardrail pattern for enterprise governance stories because it caught both permission and copy regressions.
