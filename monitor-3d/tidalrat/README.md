# tidalrat

3D views of the same mactop data tidemark shows. Rust + Ratatui, and the pilot for a Rust rewrite.

```sh
cargo install --path monitor-3d/tidalrat   # puts tidalrat in ~/.cargo/bin
tidalrat --cores                           # live mactop
tidalrat --cores --replay                  # bundled 90 s recording, no mactop needed
```

`--cores` draws per-core load as a rotating 3D bar chart in braille, inside a heavy Monitor TUI frame. Inside
[Ratty](https://blog.orhun.dev/introducing-ratty/) it's detected automatically, and the columns become real 3D
cubes anchored to each core's label cell (`--ratty` forces this; `r` toggles it).

Keys: `←/→` rotate · `↑/↓` tilt · `space` pause · `a` auto-spin · `r` ratty · `q` quit. Minimum 60×20.
