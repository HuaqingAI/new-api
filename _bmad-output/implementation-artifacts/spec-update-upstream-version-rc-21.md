---
title: 'Update upstream version to v1.0.0-rc.21'
type: 'chore'
created: '2026-07-13'
status: 'done'
route: 'one-shot'
---

# Update upstream version to v1.0.0-rc.21

## Intent

**Problem:** `UPSTREAM_VERSION` still declared `v1.0.0-rc.20` after the official `v1.0.0-rc.21` tag was merged, so release metadata would report the previous upstream version.

**Approach:** Update the single-line version pointer to `v1.0.0-rc.21`, preserving the repository's existing tag format.

## Suggested Review Order

- Confirm the version pointer matches the merged official tag.
  [`UPSTREAM_VERSION:1`](../../UPSTREAM_VERSION#L1)
