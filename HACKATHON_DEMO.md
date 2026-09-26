# IBM Bob 2.0 Hackathon Demo

## Problem

Debugging an unfamiliar repository can require repeated manual searching before a developer can safely identify the relevant code and verify a fix.

## Solution

SourceEvidence acts as a deterministic evidence layer. It scans a Git repository, analyzes dependencies and history, identifies hotspots, and packages the findings into an Evidence Pack. IBM Bob 2.0 then uses that evidence together with full repository context to investigate and implement a fix.

## Demonstrated workflow

Developer problem
→ SourceEvidence Evidence Pack
→ IBM Bob 2.0 investigation
→ minimal implementation
→ regression test
→ verification

## Reproducible example

The included `hackathon_demo/` project contains a payment retry/idempotency bug.

Before the Bob-assisted fix:

```text
go test ./...
FAIL
duplicate charge: expected total 1000, got 2000
```

Bob identified the faulty retry guard in `payment.go`, changed it so an already-processed payment ID cannot be charged again, and the same regression test then passed:

```text
go test ./...
ok demo-payment
```

## IBM Bob usage

Bob was used directly on the repository to:
- inspect the repository and the SourceEvidence evidence artifact;
- identify the root cause;
- propose the minimal fix;
- modify `payment.go`;
- verify the regression test.

Screenshots are stored in `docs/bob-session/`.
