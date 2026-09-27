A multi-row area graph drawn in braille, two samples per cell and four dots per row.

- Use for cpu, gpu and power history in the larger panels.
- You supply: values, width, height, max, and optionally a fixed role.
- With no role, each row takes the `level()` colour of its height, so the graph shades green to red as it climbs. `gpu` passes `series-gpu` and `power` passes `series-power`.
- A value above zero always shows at least one dot.
- For the power graph, label the max (`36.6W`) in `dim` on the first row, 7 cells wide.
