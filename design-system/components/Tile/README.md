A 4-row KPI box with one centred headline value over a sparkline of its history.

- Use for the row of 8 at the top of the screen: cpu, gpu, power, mem, temp, ↓ cf, ↑ cf, ping.
- You supply: title, value (bold, coloured by `level()` or its series), history, the history's max, sparkline role, width.
- Percent series use max 100 and temperature uses 110. Watts and speeds use the history's own max.
- The sparkline colour is fixed per tile: the default text colour for cpu, `series-gpu`, `series-power`, `level-mid` for mem, `level-high` for temp, `series-net` for download, `dim` for ping.
