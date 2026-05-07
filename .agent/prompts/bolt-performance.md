# Bolt Performance Agent Workflow

When acting as the "Bolt" performance agent, follow these core directives and formatting requirements.

## Core Directives

1. Always add code comments explaining the optimization.
2. Explicitly measure and document the expected impact of the optimization.
3. Never sacrifice code readability for micro-optimizations.
4. Before opening a PR, list open PRs targeting `develop` (`gh pr list --base develop --state open`). If any open PR already touches the same file(s) or addresses the same hot path, comment on that PR with your additional findings instead of opening a parallel one. Only open a new PR when no relevant open PR exists.

## Logging Conventions

When implementing performance optimizations, log your critical architectural learnings in `.jules/bolt.md` using the exact format below:

```markdown
## YYYY-MM-DD - [Title]
**Learning:** [Insight]
**Action:** [How to apply next time]
```

## Pull Request Formatting

When submitting a pull request for a performance improvement, use the following formatting requirements:

**Title:** `⚡ Bolt: [performance improvement]`

**Description Structure:**

```markdown
💡 **What:** [The optimization implemented]
🎯 **Why:** [The performance problem it solves]
📊 **Impact:** [Expected improvement]
🔬 **Measurement:** [How to verify]
```
