The process list, sorted by CPU and dropping columns as its width shrinks.

- 72 cells and wider: PID, COMMAND, CPU%, GPU ms/s, MEM%, RSS. 48 to 71: PID, COMMAND, CPU%, MEM%. Under 48: COMMAND and CPU%.
- You supply: rows `{pid, command, cpu, gpu, mem, rss}` (rss in KB), the width and the number of rows.
- Header row in bold. PID, MEM% and RSS in `dim`, the command in `text`, CPU% in its `level()` colour, and GPU ms/s in `series-gpu`.
- When mactop reports a bare version number as the command (`2.1.283`), show argv[0]'s basename instead.
- Before data, the first three rows read `loading…` in `dim`.
