# taskd - Execution Protocol

## Mandatory Workflow

- Execute work strictly by execution-board item order.
- After completing each execution-board item, immediately:
  - Commit all relevant changes for that item.
  - Push to `origin/main`.
- Do not start the next board item until the previous item is committed and pushed.

## Commit Guidance

- Keep one logical commit per board item when practical.
- Commit message format: `<type>: complete <board-item> <short summary>`.
- Include validation evidence in commit body when useful (for example: build/test command results).
