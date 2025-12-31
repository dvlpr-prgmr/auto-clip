# Development Plan

## Status
- [x] Plan initialized (2025-12-31)

## Phase 1: Core Backend Pipeline
- [x] Render clips to disk and return JSON metadata.
- [x] Generate a thumbnail per clip output.
- [x] Add `/api/preview` to fetch YouTube metadata.
- [x] Add multi-clip rendering (batch segments) endpoint.
- [x] Generate per-segment thumbnails at timestamps.

## Phase 2: Editor UI + Workflow
- [x] Editor-style layout and panels.
- [x] Auto-preview metadata on URL input.
- [x] Auto-split UI with queue rendering.
- [x] Manual segment add/edit/delete.
- [ ] Segment reordering and inline time edits.
- [ ] Wire output settings to backend payload.

## Phase 3: Reliability + Ops
- [x] Proxy/cookie handling and redacted logging.
- [ ] Add automated tests for core handlers.
- [ ] Document runtime envs and troubleshooting.

## Phase 4: Release + QA
- [ ] End-to-end smoke checklist.
- [ ] Performance checks for large videos.
