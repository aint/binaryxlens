# AGENTS.md

**Strictly follow every instruction in this file.** If a rule does not apply, say so **before** acting and ask the user — never skip silently.

## 0. General Chat & Tone
- Limit your replies to 50 words max.
- Use a straight, honest, and direct conversation style. Do not apologize.
- Drop filler words (just/really/basically/actually/simply), pleasantries (sure/certainly/of course/happy to), and hedging.
- Use short synonyms (e.g., "big" not "extensive", "fix" not "implement a solution for") and keep technical terms exact.

## 1. Core Principles & Scope Limits
- Do not expand the scope. Changes must only touch what is strictly necessary.
- If you identify out-of-scope improvements, list them exclusively as suggestions in your text response.
- **First action on any implementation task:** estimate scope and print preflight: `Preflight: ~N files | schema: yes/no | routing: yes/no | architecture: yes/no`.
- If a task impacts more than 3 files or alters core architecture, halt execution and enter **Plan Mode** to create a step-by-step execution plan for approval.
- Don't over-engineer. Make every change as simple as possible while maintaining staff engineer standards.

**On halt:** do not edit files. Use `SwitchMode` (plan), post a short step-by-step plan, wait for explicit user OK, then implement.

## 2. Subagent Strategy
- Use subagents liberally to keep the main context window clean.
- Offload research, exploration, and parallel analysis to subagents.
- For complex problems, allocate more compute via subagents.
- Assign exactly one task per subagent for focused execution. Summarize their findings back into the main context.

## 3. Self-Improvement Loop
- After ANY correction from the user, immediately create or update `lessons.md` using the following exact pattern:
  ```markdown
  ### [Topic] - [Date]
  - **Context:** What was being attempted.
  - **Mistake:** What went wrong / What the user had to correct.
  - **Fix/Lesson:** How to do it correctly next time.
  ```
- Review `lessons.md` at the start of every session for relevant project context.

## 4. Coding & Refactoring Workflow
- Ask yourself: "Would a staff engineer approve this code?"
- **Deduplication Check:** Before writing a new utility function, component, or helper, search the codebase for similar implementations. Prioritize extending existing code over creating duplicates.
- **Comment Retention:** Always preserve existing comments and documentation blocks. Do not strip code comments when modifying lines.
- **New Code Comments:** Write comments only for non-trivial or highly complex logic (e.g., a complex DFS function). Keep them sparse and meaningful.
- **Escalation Rule:** Do not introduce breaking API changes, schema changes, or dependency alterations without explicit human approval.

## 5. Escalation Rules (Banned Behaviors)
If blocked by a recurring terminal error or test failure, you are strictly **FORBIDDEN** from:
- Deleting lockfiles.
- Bypassing pre-commit hooks or using `--no-verify`.
- Commenting out failing tests to force a green build.
*Escalation Path:* If a fix fails twice consecutively, stop coding, output the exact error logs, and ask the user for direction.

## 6. Git Workflow [Scope: Git Operations]
- Write concise commit messages focused on intent and impact (problem solved, behavior achieved), not implementation details.
- **Example:** `Add cache layer` (Good) vs `Create redis.go and init client map` (Bad).

## 7. Agent Skills (On-Demand Context)
- You have access to specialized workspace skills located in `.agents/skills/`.
- Read the frontmatter descriptions of these skills at session startup.
- Automatically activate and apply the rules inside a skill (such as `.agents/skills/golang-dev/SKILL.md`) when the current task, prompt keywords, or modified files match that skill's target environment.