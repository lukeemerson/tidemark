# Spec: alert markers on the terrain (3D, tidalrat)

`tidalrat --terrain` marks where a diagnosis rule started firing, like tidemark's `▲` ticks.

## 1. Rules

- tidalrat runs the three rules from SPEC.md §3 itself, in Rust. It uses the same thresholds and priority: throttle (`thermal_state` ≠ `Nominal`); swap (+256 MB over 10 samples while pressure ≥ warn); runaway (top process ≥ 100% CPU for 10 samples). Each clears after 5 quiet samples.
- A marker is placed when a rule starts firing, not on every sample while it holds.
- Memory pressure comes from sysctl (`kern.memorystatus_vm_pressure_level`) when live. On `--replay` and `--play` there is none, so the swap rule can't fire, the same as tidemark's `-play`.
- tidemark's Go rules are the reference. Review checks that both fire on the same samples of the bundled recording: runaway at samples 10 and 56.

## 2. Markers on the terrain

- `▲` in `level-high` on the floor plane, 2.5 lane widths outside whichever side edge of the floor faces screen-left at the current yaw (x = ±(half_x + 2.5)), level with the time row where the rule fired. The markers switch sides as the terrain rotates past side-on.
- It's anchored in 3D, so it turns and tilts with the terrain.
- Always drawn on top, like the core labels. It's never skipped for lack of space.
- Only alerts inside the 90-sample window get a marker. Older ones show only as track ticks.
- When the cursor (the front row) is on an alert's sample, that marker is in `level-mid`, the cursor colour, as on tidemark's track.

## 3. Header

- The track shows `▲` ticks where rules fired: `level-high`, or `level-mid` in the cursor's cell.
- After the time comes `  ·  N alerts`, counting alerts in the 400-sample history.
- When the header narrows, the alert count drops first, before the core facts.

## 4. Also in this build

- Remove the back-edge `t−Ns` floor label. It never found room to draw, and the header already shows t−Ns for the cursor (Luke).

## 5. Out of scope

- A diagnosis line or rule text in tidalrat.
- Markers in `--cores`.
- Reading tidemark's disk store.
- Rule settings.

## Judgment calls (design)

- **Beside the floor, not on it:** the front lanes hide the floor's left edge from behind (checked in a render at yaw 0, pitch 0.45). 2.5 lanes out lands in clear space. At the default yaw 0.6 the fixed left side swung behind the terrain for old alerts (build-3d's capture), so the side follows the view (Luke).
- **`▲` only, no label:** several alerts in the window would crowd the terrain. The rule text belongs to a later diagnosis line.
