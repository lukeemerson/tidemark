# Future builds: 3D versions of history and diagnosis

Not in v1. These build on SPEC.md's recording format and rules.

- [ ] **Throttle hotspot** (`--die`, prototype B): the die floorplan glows by temperature while the thermal rule fires.
- [ ] **Runaway process column**: the runaway process as a raised LoadColumn over the core grid.
- [ ] **Compare in 3D**: two terrains side by side (the 2D version is COMPARE-SPEC.md).
- [ ] **Thermal state names**: record mactop under sustained load until `thermal_state` leaves `Nominal`. Replace the `‹STATE›` placeholder in SPEC.md rule 1 and the mockups with the real names.
- [ ] **Process actions** (2D): kill/renice from the runaway diagnosis, with a y/n confirm.
- [ ] **tidalrat reads the disk store**: `--terrain` over the 24 h summary tier (STORE-SPEC.md).
- [ ] **Pixel-rendered 3D via terminal image protocols**: `--cores` and `--terrain` drawn as images through ratatui-image (Kitty graphics in Ghostty/Kitty/WezTerm, the iTerm2 protocol, Sixel), falling back to braille where unsupported (Alacritty). Escape payloads are written after draw, never put in buffer cells (the root cause found in `ratty-stale-fix`).
- [ ] **Ratty**: real 3D objects inside Ratty via RGP. Removed from tidalrat for now; detection was never checked against a real Ratty.
