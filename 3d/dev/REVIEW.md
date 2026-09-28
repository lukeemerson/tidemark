# Review: what exists before the 3D work

| piece | where | state |
| --- | --- | --- |
| Monitor | `bin/monitor` | zsh, draft 03 layout, mactop + saved cloudy runs. Committed (1709a90). |
| Drafts | `drafts/v01–v10.zsh`, `drafts/core.zsh` | Ten framings on a shared engine. Kept for reference. |
| Renderer | `render/` | pyte → HTML captures; `mactop.raw` = 90 recorded samples. |
| Go port plan | `PLAN.md` | Bubble Tea port. Not started. |
| Design system | `design-system/` (commit cf8e78f) and the Monitor TUI artifact | Tokens, brand book, 10 components as a JS port of the zsh draw functions. |

## What holds up
- The token roles (`level-*`, `series-*`, `dim`) cover every colour decision in all ten drafts. That makes them a good contract for a 3D view too.
- The JS port (`design-system/components/bundle.js`) produces the same frames as the zsh, so a browser prototype can draw the real TUI instead of imitating it.
- Before data arrives, every value shows as a dash, so the layout never shifts. That carries straight into 3D: draw the geometry first, animate the values in later.

## Gaps and risks
- The design system lives in two copies: the artifact and `design-system/`. Nothing syncs them.
- `mactop.raw` still names processes by version (`2.1.283`). `bin/monitor` resolves those names, but the prototypes replay the raw data, so they'll show the version strings.
- The cloudflare figures in the prototypes are fixed values copied from the v03 capture, not from saved runs.
- There's no real-time path outside zsh yet (the Go port hasn't started). Every prototype here replays recorded data.
