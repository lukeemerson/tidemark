// C · History terrain: cores (x) × last 90 samples (z, newest at front), height = load.
(function () {
  var N = 90, C = 10, DZ = 0.2, HT = 4, XS = 1.2;
  var NAMES = ['e1', 'e2', 'e3', 'e4', 'p1', 'p2', 'p3', 'p4', 'p5', 'p6'];
  var SER = [{ name: 'cpu', key: 'cpu', max: 100, unit: '%' }, { name: 'gpu', key: 'gpu', max: 100, unit: '%', tok: 'series-gpu' },
    { name: 'power', key: 'pw', max: 40, unit: ' W', tok: 'series-power' }];
  var hist = null, mode = 'cores', off = 0, iso = null, lastE = null, hovering = false;
  var bc = BC({
    title: [['', SAMPLES.meta.name, true], ['dim', '  ·  history terrain  ·  last 90 s']],
    presets: [[[9, 13, 14], [0, 0, -7]], [[0, 5, 10], [0, 1, -8.9]], [[0, 26, -8.8], [0, 0, -8.9]], [[20, 6, -8.9], [0, 1, -8.9]]],
    legend: [['l', 'layer: cores / cpu+gpu+power'], ['◀ ▶', 'drag to scrub · ← → while paused']],
    onSample: function (s, h) { hist = h; draw(); if (hovering) onHover(lastE); },
    onArrow: function (d) { setOff(off - d); }, onKey: function (k) { if (k === 'l') { mode = mode === 'cores' ? 'series' : 'cores'; iso = null; draw(); } },
    onHover: onHover, onClick: onClick
  });
  var T = bc.T, M = bc.M, scene = bc.scene, col = new T.Color();
  function Z(j) { return (j - (N - 1)) * DZ; }
  function X(c) { return (c - (C - 1) / 2) * XS; }
  function win(a) { var w = a.slice(-N); while (w.length < N) w.unshift(null); return w; }
  function lineMat() { return new T.LineBasicMaterial({ vertexColors: true, transparent: true }); }
  function attr(g, name, n) { g.setAttribute(name, new T.BufferAttribute(new Float32Array(n * 3), 3)); }

  // floor grid in dim: one line per 10 s, plus the side rails
  var fp = [], x0 = X(0) - 0.8, x1 = X(C - 1) + 0.8, j;
  for (j = N - 1; j >= 0; j -= 10) fp.push(x0, 0, Z(j), x1, 0, Z(j));
  fp.push(x0, 0, Z(0), x0, 0, 0, x1, 0, Z(0), x1, 0, 0);
  var fg = new T.BufferGeometry();
  fg.setAttribute('position', new T.Float32BufferAttribute(fp, 3));
  scene.add(new T.LineSegments(fg, new T.LineBasicMaterial({ color: THEME.dim })));

  // Terrain: one lane per core, each an extruded area chart (top + two side walls) with a
  // flat level colour per 1-sample segment, so the level bands stay readable
  var LW = 0.42, QV = 18, tg = new T.BufferGeometry(), c;
  attr(tg, 'position', C * (N - 1) * QV); attr(tg, 'color', C * (N - 1) * QV);
  var terrain = new T.Mesh(tg, new T.MeshLambertMaterial({ vertexColors: true, side: T.DoubleSide }));
  scene.add(terrain);
  var ridges = NAMES.map(function () {
    var g = new T.BufferGeometry(); attr(g, 'position', N); attr(g, 'color', N);
    var l = new T.Line(g, lineMat()); scene.add(l); return l;
  });

  // series ribbons: unlit vertical strips at x = -3, 0, 3
  var ribbons = SER.map(function (sr, k) {
    var g = new T.BufferGeometry(), ix = [];
    attr(g, 'position', N * 2); attr(g, 'color', N * 2);
    for (j = 0; j < N - 1; j++) ix.push(2 * j, 2 * j + 2, 2 * j + 1, 2 * j + 1, 2 * j + 2, 2 * j + 3);
    g.setIndex(ix);
    var m = new T.Mesh(g, new T.MeshBasicMaterial({ vertexColors: true, side: T.DoubleSide, transparent: true }));
    m.userData.k = k; m.position.x = (k - 1) * 3; scene.add(m); return m;
  });

  // highlighted time slice
  var sg = new T.BufferGeometry(); attr(sg, 'position', C);
  var slice = new T.Line(sg, new T.LineBasicMaterial({ color: THEME['text-bright'], depthTest: false }));
  scene.add(slice);

  var coreLbl = NAMES.map(function (n, k) { var l = bc.label(new T.Vector3(X(k), 0, 0.6), [[['dim', n]]]); return l; });
  var serLbl = SER.map(function (sr, k) { return bc.label(new T.Vector3((k - 1) * 3, 0, 0.6), [[['dim', sr.name]]]); });

  function cores() { return win(hist.cores); }
  function series(k) { return win(hist[SER[k].key]); }
  function setCol(arr, i, hex) { col.set(hex); arr[i * 3] = col.r; arr[i * 3 + 1] = col.g; arr[i * 3 + 2] = col.b; }

  function draw() {
    if (!hist) return;
    var cs = cores(), cm = mode === 'cores', k;
    terrain.visible = cm;
    ridges.forEach(function (r, c) { r.visible = cm && (iso === null || iso === c); });
    ribbons.forEach(function (r, k) { r.visible = !cm; r.material.opacity = iso === null || iso === k ? 1 : 0.15; });
    coreLbl.forEach(function (l) { l.el.dataset.hide = cm ? '' : '1'; });
    serLbl.forEach(function (l) { l.el.dataset.hide = cm ? '1' : ''; });
    var sp = sg.attributes.position.array;
    if (cm) {
      var p = tg.attributes.position.array, q = tg.attributes.color.array, n = 0;
      var H = function (jj) { return (cs[jj] ? cs[jj][c] : 0) / 100 * HT + 0.02; };
      var put = function (x, y, z, hex) { p[n * 3] = x; p[n * 3 + 1] = y; p[n * 3 + 2] = z; setCol(q, n, hex); n++; };
      var quad = function (a, b, cc, d, hex) { put.apply(0, a.concat(hex)); put.apply(0, b.concat(hex)); put.apply(0, cc.concat(hex)); put.apply(0, a.concat(hex)); put.apply(0, cc.concat(hex)); put.apply(0, d.concat(hex)); };
      for (c = 0; c < C; c++) for (j = 0; j < N - 1; j++) {
        var xl = X(c) - LW, xr = X(c) + LW, z0 = Z(j), z1 = Z(j + 1), y0 = H(j), y1 = H(j + 1);
        var seg = cs[j + 1] ? cs[j + 1][c] : null;
        var hex = seg == null || (iso !== null && iso !== c) ? THEME.dim : bc.lvl(seg);
        quad([xl, y0, z0], [xr, y0, z0], [xr, y1, z1], [xl, y1, z1], hex);
        quad([xl, 0, z0], [xl, y0, z0], [xl, y1, z1], [xl, 0, z1], hex);
        quad([xr, 0, z0], [xr, y0, z0], [xr, y1, z1], [xr, 0, z1], hex);
      }
      for (j = 0; j < N; j++) for (c = 0; c < C; c++) {
        var v = cs[j] ? cs[j][c] : 0, y = v / 100 * HT + 0.02;
        var rp = ridges[c].geometry.attributes.position.array, rq = ridges[c].geometry.attributes.color.array;
        rp[j * 3] = X(c); rp[j * 3 + 1] = y + 0.03; rp[j * 3 + 2] = Z(j);
        setCol(rq, j, cs[j] ? bc.lvl(v) : THEME.dim);
      }
      tg.attributes.position.needsUpdate = tg.attributes.color.needsUpdate = true;
      tg.computeVertexNormals(); tg.computeBoundingSphere();
      ridges.forEach(function (r) { r.geometry.attributes.position.needsUpdate = r.geometry.attributes.color.needsUpdate = true; r.geometry.computeBoundingSphere(); });
      var js = N - 1 - off, row = cs[js];
      for (c = 0; c < C; c++) { sp[c * 3] = X(c); sp[c * 3 + 1] = (row ? row[c] : 0) / 100 * HT + 0.06; sp[c * 3 + 2] = Z(js); }
      sg.setDrawRange(0, C);
    } else {
      for (k = 0; k < SER.length; k++) {
        var vs = series(k), g = ribbons[k].geometry, rp2 = g.attributes.position.array, rc = g.attributes.color.array;
        for (j = 0; j < N; j++) {
          var h = (vs[j] || 0) / SER[k].max * HT, tok = vs[j] == null ? THEME.dim : SER[k].tok ? THEME[SER[k].tok] : bc.lvl(vs[j]);
          rp2.set([0, 0, Z(j), 0, h, Z(j)], j * 6);
          setCol(rc, 2 * j, tok); setCol(rc, 2 * j + 1, tok);
        }
        g.attributes.position.needsUpdate = g.attributes.color.needsUpdate = true; g.computeBoundingSphere();
        var jv = series(k)[N - 1 - off];
        sp[k * 3] = (k - 1) * 3; sp[k * 3 + 1] = (jv || 0) / SER[k].max * HT + 0.06; sp[k * 3 + 2] = Z(N - 1 - off);
      }
      sg.setDrawRange(0, SER.length);
    }
    sg.attributes.position.needsUpdate = true; sg.computeBoundingSphere();
    scrubber();
  }

  // Scrubber in TUI glyphs
  var W = 30, sc = bc.pre('scrub');
  sc.style.cssText = 'right:0;bottom:0;pointer-events:auto;cursor:ew-resize;touch-action:none';
  function scrubber() {
    var f = Math.round((N - 1 - off) / (N - 1) * W);
    M.Screen(sc, [[['dim', '◀ '], ['', '▮'.repeat(f)], ['dim', '▯'.repeat(W - f) + ' ▶  '], ['', 't−' + off + 's', true]]]);
  }
  function setOff(o) { off = Math.max(0, Math.min(N - 1, o)); draw(); }
  function scrubAt(e) {
    var r = sc.getBoundingClientRect(), pad = parseFloat(getComputedStyle(sc).paddingLeft);
    var cw = (r.width - 2 * pad) / sc.textContent.length, f = (e.clientX - r.left - pad - 2 * cw) / (W * cw);
    setOff(Math.round((1 - Math.max(0, Math.min(1, f))) * (N - 1)));
  }
  var drag = false;
  sc.addEventListener('pointerdown', function (e) { drag = true; sc.setPointerCapture(e.pointerId); scrubAt(e); });
  sc.addEventListener('pointermove', function (e) { if (drag) scrubAt(e); });
  sc.addEventListener('pointerup', function () { drag = false; });

  // hover + click on the terrain / ribbons
  function hit(e) {
    var h = bc.pick(e, mode === 'cores' ? [terrain] : ribbons);
    if (!h) return null;
    var jj = Math.max(0, Math.min(N - 1, Math.round(h.point.z / DZ + N - 1)));
    return mode === 'cores' ? { k: Math.max(0, Math.min(C - 1, Math.round(h.point.x / XS + (C - 1) / 2))), j: jj } : { k: h.object.userData.k, j: jj };
  }
  function onHover(e) {
    lastE = e;
    var h = e && hist && hit(e), v, line;
    hovering = !!h;
    if (!h) { bc.tip(null); return; }
    if (mode === 'cores') { v = cores()[h.j]; v = v ? v[h.k] : null; line = [[v == null ? 'dim' : M.level(v), v == null ? '—' : v.toFixed(0) + '%', true]]; }
    else {
      var sr = SER[h.k]; v = series(h.k)[h.j];
      line = [[v == null ? 'dim' : sr.tok || M.level(v), v == null ? '—' : v.toFixed(sr.key === 'pw' ? 1 : 0) + sr.unit, true]];
    }
    bc.tip(e, mode === 'cores' ? NAMES[h.k] : SER[h.k].name, [line.concat([['dim', '  t−' + (N - 1 - h.j) + 's']])]);
  }
  function onClick(e) { var h = hit(e); iso = h && h.k !== iso ? h.k : null; draw(); }

  bc.start();
})();
