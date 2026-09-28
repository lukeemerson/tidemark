// B · Silicon floor: a schematic M2 Pro floorplan of DieBlocks that extrude with load and glow with temperature.
(function () {
  var S = SAMPLES.samples, m = SAMPLES.meta, blocks = [], hover = null, lastE = null, sel = null, cur = S[0];
  var bc = BC({
    title: [['', m.name, true], ['dim', '  ·  schematic floorplan  ·  4E + 6P CPU  ·  16-core GPU']],
    presets: [[[11.6, 11.5, 11.6], [0, 0, 0]], [[0, 22, 0.01], [0, 0, 0]], [[0, 6, 18], [0, 0, 0]], [[18, 6, 0], [0, 0, 0]]],
    onSample: update, onHover: onHover, onClick: onClick
  });
  var T = bc.T, M = bc.M, board = new T.Group();
  board.position.set(-1, 0, -1.5);
  board.scale.z = -1; // CPU row in front, tall memory block behind it
  bc.scene.add(board);
  var dimMat = new T.LineBasicMaterial({ color: THEME.dim });

  function outline(x0, z0, x1, z1) {
    var g = new T.BufferGeometry().setFromPoints([[x0, z0], [x1, z0], [x1, z1], [x0, z1], [x0, z0]].map(function (p) { return new T.Vector3(p[0], 0.01, p[1]); }));
    board.add(new T.Line(g, dimMat));
  }
  function block(kind, idx, name, x, z, w, d) {
    var geo = new T.BoxGeometry(w, 1, d).translate(0, 0.5, 0);
    var mat = new T.MeshLambertMaterial({ color: THEME['level-low'], transparent: true });
    var mesh = new T.Mesh(geo, mat), edges = new T.LineSegments(new T.EdgesGeometry(geo), dimMat.clone());
    edges.material.transparent = true;
    mesh.add(edges);
    mesh.position.set(x, 0, z);
    board.add(mesh);
    var b = { kind: kind, idx: idx, name: name, mesh: mesh, edges: edges, h: 0.1, to: 0.1, load: 0 };
    mesh.userData.b = b;
    blocks.push(b);
  }
  function label(x, z) { return bc.label(new T.Vector3(x - 1, 0, -1.5 - z), [[]]); }

  var k;
  for (k = 0; k < 4; k++) block('e', k, 'e' + (k + 1), -7 + k * 1.4, -3, 1.1, 1.1);
  for (k = 0; k < 6; k++) block('p', k, 'p' + (k + 1), -0.6 + k * 1.8, -3, 1.5, 1.5);
  for (k = 0; k < 16; k++) block('g', k, 'gpu ' + (k + 1), -7 + (k % 8) * 1.1, 0 + Math.floor(k / 8) * 1.1, 0.9, 0.9);
  block('m', 0, 'memory', 5.2, 0.55, 6, 2);
  outline(-7.8, -4.1, -1.9, -2.1); outline(-1.6, -4.1, 9.8, -1.9); outline(-7.8, -0.8, 1.5, 1.9); outline(1.9, -0.8, 8.6, 1.9);
  outline(-8.4, -5.6, 10.4, 2.5);
  var L = { e: label(-4.85, -4.7), p: label(4.1, -4.7), g: label(-3.15, -1.4), m: label(5.25, -1.4) };

  function gb(n) { return (n / 1073741824).toFixed(1); }
  // deterministic per-GPU-core spread around sample.gpu
  function gpuLoad(s, i, c) { return Math.max(0, Math.min(100, s.gpu * (1 + 0.35 * Math.sin(c * 2.39 + i * 0.7)))); }
  // the recording spans 61–69°, so the glow is scaled over 55–75° to make that range visible
  function emis(t) { return 0.05 + 0.4 * Math.max(0, Math.min(1, (t - 55) / 20)); }

  function update(s, hist, i) {
    cur = s;
    blocks.forEach(function (b) {
      var load, colr, t;
      if (b.kind === 'e' || b.kind === 'p') { load = s.cores[b.kind === 'e' ? b.idx : 4 + b.idx]; colr = bc.lvl(load); t = s.tc; }
      else if (b.kind === 'g') { load = gpuLoad(s, i, b.idx); colr = THEME['series-gpu']; t = s.tg; }
      else { load = s.mu * 100 / s.mt; colr = bc.lvl(load); t = null; }
      b.load = load;
      // memory is capacity, not activity: a low slab so it can't hide the cores
      b.to = 0.1 + load / 100 * (b.kind === 'm' ? 0.8 : 3);
      b.mesh.material.color.set(colr);
      b.mesh.material.emissive.set(colr);
      b.mesh.material.emissiveIntensity = t == null ? 0.05 : emis(t);
    });
    M.Screen(L.e.el, [[['', 'e-cluster', true], ['dim', '  ' + s.efreq + ' MHz']]]);
    M.Screen(L.p.el, [[['', 'p-cluster', true], ['dim', '  ' + s.pfreq + ' MHz']]]);
    M.Screen(L.g.el, [[['', 'gpu', true], ['dim', '  ' + s.gfreq + ' MHz  ·  ' + s.tg.toFixed(0) + '°']]]);
    M.Screen(L.m.el, [[['', 'memory', true], ['dim', '  ' + gb(s.mu) + ' / ' + gb(s.mt) + ' GB']]]);
    if (hover) showTip(lastE, hover);
  }

  function showTip(e, b) {
    var s = cur, pct = b.load.toFixed(0) + '%', line;
    if (b.kind === 'e') line = [[M.level(b.load), pct, true], ['dim', '  e-cluster ' + s.efreq + ' MHz']];
    else if (b.kind === 'p') line = [[M.level(b.load), pct, true], ['dim', '  p-cluster ' + s.pfreq + ' MHz']];
    else if (b.kind === 'g') line = [['series-gpu', pct, true], ['dim', '  gpu ' + s.gfreq + ' MHz']];
    else line = [[M.level(b.load), pct, true], ['dim', '  ' + gb(s.mu) + ' / ' + gb(s.mt) + ' GB']];
    bc.tip(e, b.name, [line]);
  }
  function hit(e) { var h = bc.pick(e, blocks.map(function (b) { return b.mesh; })); return h && h.object.userData.b; }
  function onHover(e) {
    lastE = e;
    hover = hit(e);
    if (hover) showTip(e, hover); else bc.tip(null);
  }
  function onClick(e) {
    var b = hit(e);
    sel = b && b !== sel ? b : null;
    blocks.forEach(function (o) {
      var op = sel && o !== sel ? 0.18 : 1;
      o.mesh.material.opacity = op;
      o.edges.material.opacity = op;
    });
  }

  bc.frame(function (dt) {
    var a = 1 - Math.exp(-dt / 0.1); // ~400 ms to settle
    blocks.forEach(function (b) { b.h += (b.to - b.h) * a; b.mesh.scale.y = b.h; });
  });
  bc.start();
})();
