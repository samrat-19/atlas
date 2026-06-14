# Atlas Decision Log

This document records decisions that affect multiple phases or constrain the
repository model. Plans describe intended work; this log records what was
actually decided and why.

## Decision Template

### DEC-000: Decision title

- Date:
- Status: Proposed, accepted, superseded, or rejected
- Phase:
- Context:
- Decision:
- Alternatives considered:
- Consequences:
- Follow-up:

## Open Phase 1 Decisions

The following decisions are currently proposed but not accepted:

- Whether canonical snapshots include any root identity.
- Whether canonical JSON becomes a supported public contract in Phase 1.
- Whether default scans prune `node_modules` and Bazel output entirely.
- Whether counts inside pruned directories remain unknown or are measured using
  a lightweight secondary strategy.
- Whether inaccessible paths produce partial results by default.
- Whether Git ignore behavior is opt-in or deferred.
- Whether the CLI moves immediately to `atlas scan <path>`.
- Whether current module inference is labeled as provisional or legacy in the
  new snapshot.
