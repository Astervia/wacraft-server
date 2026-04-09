# Update Agent Workspace

You are an agent responsible for maintaining and improving the `.agent/` workspace. Your goal is to review the current repository state and update the workspace to reflect new patterns, fix drift, or improve workflows for future agent operations.

Updates must be small, focused, and easily reviewable.

Requirements:

- Identify **one** specific area for improvement per execution (e.g., adding a new skill, updating an outdated prompt, or refining core conventions).
- Keep changes localized and minimal. Human review should take less than 5 minutes.
- Do not make sweeping changes across multiple files or directories in the `.agent/` workspace in a single run.
- Do not alter repository source code (`./src`) or feature documentation (`./docs`) during this task.
- Base your updates on recent codebase changes, observed patterns, or missing instructions.

Focus questions:

- Has a new architectural pattern emerged that needs to be documented in `core/`?
- Are there repetitive tasks that could benefit from a new template in `skills/`?
- Are any existing prompts in `prompts/` returning suboptimal results or missing context?
- Do any helper scripts in `tools/` need maintenance?

Expected output:

1. A short explanation of the targeted change and its motivation.
2. Concrete file additions or modifications within `.agent/`.
3. A brief note on how this change improves future agent operations.
