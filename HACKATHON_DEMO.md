# IBM Bob 2.0 Hackathon Demo

## Problem

Debugging an unfamiliar repository can require repeated manual searching before a developer can safely identify the relevant code and verify a fix.

## Solution

SourceEvidence acts as a deterministic evidence layer. It scans a Git repository, analyzes dependencies and history, identifies hotspots, and packages the findings into an Evidence Pack.

IBM Bob 2.0 then uses that evidence together with full repository context to investigate the problem, plan the change, and implement the fix.

## Demonstrated Workflow

```text
Developer problem
        |
        v
SourceEvidence Evidence Pack
        |
        v
IBM Bob 2.0 investigation
        |
        v
Minimal implementation
        |
        v
Regression test
        |
        v
Verification
```

The roles remain separate: SourceEvidence provides deterministic repository evidence, IBM Bob 2.0 provides repository-aware reasoning and development actions, and the developer reviews the resulting change.

## Reproducible Example

The included `hackathon_demo/` project contains a payment retry/idempotency bug.

Before the Bob-assisted fix:

```text
go test ./...
FAIL
duplicate charge: expected total 1000, got 2000
```

The failure demonstrates that retrying the same payment can incorrectly result in a second charge.

Bob identified the faulty retry guard in `payment.go` and applied the minimal correction so an already-processed payment ID cannot be charged again.

After the fix, the same regression test passed:

```text
go test ./...
ok demo-payment
```

## IBM Bob Usage

Bob was used directly on the repository to:

- inspect the repository and SourceEvidence evidence artifact;
- investigate the failing payment test;
- identify the faulty retry guard;
- propose the minimal correction;
- modify `payment.go`;
- work through the resulting test verification.

The resulting regression test was also independently run against the demo project and passed.

Screenshots documenting the Bob session are stored in:

```text
docs/bob-session/
```

The public repository also contains the complete demonstration project under:

```text
hackathon_demo/
```
