# SourceEvidence + IBM Bob 2.0 Debugging Demo

This small Go project is the reproducible debugging example used for the IBM Bob 2.0 hackathon.

## Workflow demonstrated

1. Start from a reproducible duplicate-payment retry bug.
2. SourceEvidence analyzes the Git repository and produces an Evidence Pack.
3. IBM Bob 2.0 uses the repository context plus the Evidence Pack to investigate the failure.
4. Bob identifies the faulty retry guard and applies the minimal fix.
5. The existing regression test passes after the fix.

The demo project is intentionally small. SourceEvidence itself is designed to analyze other Git repositories.
