A horizontal meter of `■` on-cells and `·` off-cells for a percentage.

- Use for memory, a volume's fill and per-core load.
- You supply: percent, width in cells, and optionally a fixed role and the on/off characters.
- On-cells take `level()` colour unless you pass a role. Off-cells are always `dim`.
- Values over 100 fill the bar. Rounding is to the nearest cell.
- In the cores panel, label each bar `E1…` or `P1…` in `dim` and follow it with the percent in the same level colour.
