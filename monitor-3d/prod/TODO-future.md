# Future builds: 3D versions of history and diagnosis

Not in v1. These build on SPEC.md's recording format and rules.

- [ ] **Alert markers in 3D**: a diagnosis rule firing leaves a `▲` marker on the terrain's time axis, like the scrubber tick.
- [ ] **Throttle hotspot** (`--die`, prototype B): the die floorplan glows by temperature while the thermal rule fires.
- [ ] **Runaway process column**: the runaway process as a raised LoadColumn over the core grid.
- [ ] **Compare in 3D**: two terrains side by side (the 2D version is COMPARE-SPEC.md).
- [ ] **Thermal state names**: record mactop under sustained load until `thermal_state` leaves `Nominal`. Replace the `‹STATE›` placeholder in SPEC.md rule 1 and the mockups with the real names.
- [ ] **Process actions** (2D): kill/renice from the runaway diagnosis, with a y/n confirm.
- [ ] **tidalrat reads the disk store**: `--terrain` over the 24 h summary tier (STORE-SPEC.md).
