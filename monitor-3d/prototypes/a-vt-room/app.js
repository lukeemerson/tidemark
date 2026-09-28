// A · VT in the room: the draft-03 screen as a texture on a bendable slab, with objects
// anchored to its cells (Ratty RGP semantics). Components follow ../../COMPONENTS.md.
(function () {
  var T = window.MonitorTUI, TH = window.THEME, R = window.Replay, M = window.SAMPLES.meta;
  var COLS = 140, ROWS = 42, W = 138, X = 1;
  var reduced = matchMedia('(prefers-reduced-motion: reduce)').matches;
  var FONT = getComputedStyle(document.documentElement).getPropertyValue('--font-mono').trim();

  // ---- layout constants (bin/monitor layout() at 140×42, XPAD 1), shared by compose and regions
  var TW = (W - 7) / 8 | 0, TLAST = W - 7 * TW - 7, CW = (W - 1) / 2 | 0, PH = ROWS - 21, PCW = W * 2 / 3 | 0;
  var TILE_NAMES = ['cpu', 'gpu', 'power', 'mem', 'temp', '↓ cf', '↑ cf', 'ping'];
  var REGIONS = [{ name: 'header', row: 0, col: X, w: W, h: 1 }];
  TILE_NAMES.forEach(function (n, k) { REGIONS.push({ name: n, row: 1, col: X + k * (TW + 1), w: k < 7 ? TW : TLAST, h: 4 }); });
  REGIONS.push(
    { name: 'cpu', row: 5, col: X, w: CW, h: 9 }, { name: 'gpu', row: 5, col: X + CW + 1, w: W - CW - 1, h: 9 },
    { name: 'cores', row: 14, col: X, w: CW, h: 7 }, { name: 'power', row: 14, col: X + CW + 1, w: W - CW - 1, h: 7 },
    { name: 'processes', row: 21, col: X, w: PCW, h: PH },
    { name: 'memory', row: 21, col: X + PCW + 1, w: W - PCW - 1, h: 5 }, { name: 'sensors', row: 26, col: X + PCW + 1, w: W - PCW - 1, h: 5 },
    { name: 'io', row: 31, col: X + PCW + 1, w: W - PCW - 1, h: 5 }, { name: 'cloudflare', row: 36, col: X + PCW + 1, w: W - PCW - 1, h: PH - 15 });
  var CORE_CW = (CW - 4 - 3) / 2 | 0, CORE_BW = CORE_CW - 8;
  function coreCell(k) { return { row: 15 + (k >> 1), col: X + 2 + (k & 1) * (CORE_CW + 3) }; }
  function coreLabel(k) { return k < M.e ? 'E' + (k + 1) : 'P' + (k + 1 - M.e); }

  // ---- composition: the draft-03 screen as MonitorTUI lines
  function pf(n, d, w) { var s = n.toFixed(d); return ' '.repeat(Math.max(0, w - s.length)) + s; }
  function hmax(a, m) { return Math.max.apply(null, [m].concat(a)); }
  function gb(b) { return (b / 1073741824).toFixed(1); }
  function rate(b) { var u = ['B', 'K', 'M', 'G'], i = 0; while (b >= 1024 && i < 3) { b /= 1024; i++; } return b.toFixed(1) + ' ' + u[i] + '/s'; }
  function vis(l) { return l.reduce(function (n, s) { return n + Array.from(s[1]).length; }, 0); }
  var DASH = ['dim', '—'];

  function compose(s, h) {
    var mp = s.mu * 100 / s.mt, cf = M.cf;
    var hc = [[T.level(s.cpu), pf(s.cpu, 0, 3) + '%', true]], hg = [['series-gpu', pf(s.gpu, 0, 3) + '%', true]];
    var hp = [['series-power', s.pw.toFixed(1) + ' W', true]], hm = [[T.level(mp), pf(mp, 0, 3) + '%', true]];
    var out = T.header({ name: M.name, e: M.e, p: M.p, gpuCores: M.gpuCores, have: true, clock: s.t.slice(11, 19) }, W);
    var tiles = [
      T.tile('cpu', hc, h.cpu, 100, '', TW), T.tile('gpu', hg, h.gpu, 100, 'series-gpu', TW),
      T.tile('power', hp, h.pw, hmax(h.pw, 1), 'series-power', TW), T.tile('mem', hm, h.mem, 100, 'level-mid', TW),
      T.tile('temp', [[T.level(s.tc), s.tc.toFixed(0) + '°', true]], h.tc, 110, 'level-high', TW),
      T.tile('↓ cf', [['series-net', cf.dl + ' Mbps', true]], cf.hist_dl, hmax(cf.hist_dl, 1), 'series-net', TW),
      T.tile('↑ cf', [['series-power', cf.ul + ' Mbps', true]], cf.hist_ul, hmax(cf.hist_ul, 1), 'series-power', TW),
      T.tile('ping', [['', cf.lat + ' ms', true]], null, 1, 'dim', TLAST)];
    out = out.concat(T.hjoin.apply(null, [1].concat(tiles)));

    var cpu = T.box('cpu', hc, CW, 9, [[['dim', ' E '], ['', pf(s.efreq, 0, 4)], ['dim', ' · P '], ['', pf(s.pfreq, 0, 4)], ['dim', ' MHz']]]
      .concat(T.graph(h.cpu, CW - 4, 6, 100)));
    var gpu = T.box('gpu', hg, W - CW - 1, 9, [[['dim', ' ' + M.gpuCores + ' cores  ·  '], ['', pf(s.gfreq, 0, 4) + ' MHz'], ['dim', '  ·  ANE '], DASH]]
      .concat(T.graph(h.gpu, W - CW - 5, 6, 100, 'series-gpu')));
    out = out.concat(T.hjoin(1, cpu, gpu));

    var cl = [];
    for (var i = 0; i < 5; i++) {
      var l = [];
      for (var j = 0; j < 2; j++) {
        var k = i * 2 + j, p = s.cores[k] || 0;
        l = l.concat([['dim', T.fit(coreLabel(k), 3)]], T.bar(p, CORE_BW), [[T.level(p), pf(p, 0, 4) + '%']]);
        if (j < 1) l.push(['', '   ']);
      }
      cl.push(l);
    }
    var pmax = hmax(h.pw, 1), pg = T.graph(h.pw, W - CW - 1 - 4 - 7, 4, pmax, 'series-power');
    var pl = [[['dim', ' system '], DASH, ['dim', ' W  ·  GPU '], ['', s.pwg.toFixed(1)], ['dim', ' W']]]
      .concat(pg.map(function (g, n) { return [['dim', n ? '       ' : pf(pmax, 1, 5) + 'W ']].concat(g); }));
    out = out.concat(T.hjoin(1, T.box('cores', null, CW, 7, cl), T.box('power', hp, W - CW - 1, 7, pl)));

    var rw = W - PCW - 1, iw = rw - 4;
    var procs = s.procs.slice().sort(function (a, b) { return b.cpu - a.cpu; })
      .map(function (p) { return { pid: p.pid, command: p.cmd, cpu: p.cpu, gpu: 0, mem: p.mem, rss: p.rss }; });
    var right = T.box('memory', hm, rw, 5, [
      T.spread([[T.level(mp), gb(s.mu)], ['dim', ' / ' + gb(s.mt) + ' GB']], [['dim', 'swap — / — GB']], iw),
      T.bar(mp, iw), [['dim', ' DRAM'], ['', '  ↓ ' + s.dr.toFixed(1) + '  ↑ ' + s.dw.toFixed(1) + ' GB/s']]])
      .concat(T.box('sensors', null, rw, 5, [
        [['dim', ' CPU '], [T.level(s.tc), s.tc.toFixed(0) + '°'], ['dim', '   GPU '], [T.level(s.tg), s.tg.toFixed(0) + '°']],
        [['dim', ' thermal '], DASH, ['dim', '   fan '], ['dim', '— rpm']]]))
      .concat(T.box('io', null, rw, 5, [
        [['dim', ' NET  '], ['', '↓ ' + rate(s.nin) + '  ↑ ' + rate(s.nout)]],
        [['dim', ' DISK '], ['', 'r ' + rate(s.dkr * 1024) + '  w ' + rate(s.dkw * 1024)]], [['dim', ' volume —']]]))
      .concat(T.box('cloudflare', null, rw, PH - 15, [
        [['series-net', '↓ ' + cf.dl, true], ['dim', ' Mbps'], ['', '   '], ['series-net', '↑ ' + cf.ul, true], ['dim', ' Mbps']],
        [['dim', ' ping '], ['', cf.lat + ' ms'], ['dim', '  jitter '], ['', cf.jit + ' ms'], ['dim', '  loss '], DASH],
        [['dim', ' loaded ↓ — ↑ — ms   bloat —  stable —']],
        [['dim', ' ↓ hist '], ['series-net', T.spark(cf.hist_dl, cf.hist_dl.length, hmax(cf.hist_dl, 1))]],
        [['dim', ' ↑ hist '], ['series-power', T.spark(cf.hist_ul, cf.hist_ul.length, hmax(cf.hist_ul, 1))]]]));
    out = out.concat(T.hjoin(1, T.box('processes', null, PCW, PH, T.processTable(procs, PCW - 4, PH - 3)), right));
    return out.map(function (l) { return [['', ' ']].concat(l); });
  }

  // ---- TermSurface: cell-grid canvas → CanvasTexture on a bendable plane
  var S = Math.max(2, devicePixelRatio || 1), FPX = 11.5 * S, CH = 11.5 * 1.18 * S, CWPX;
  var cvs = document.createElement('canvas'), ctx = cvs.getContext('2d');
  function drawScreen(lines) {
    ctx.fillStyle = TH.bg; ctx.fillRect(0, 0, cvs.width, cvs.height);
    ctx.textBaseline = 'middle'; ctx.textAlign = 'left';
    lines.forEach(function (line, r) {
      var c = 0, y = (r + 0.5) * CH;
      line.forEach(function (seg) {
        ctx.font = (seg[2] ? '700 ' : '400 ') + FPX + 'px ' + FONT;
        ctx.fillStyle = TH[seg[0]] || TH.text;
        Array.from(seg[1]).forEach(function (ch) { if (ch !== ' ') ctx.fillText(ch, c * CWPX, y); c++; });
      });
    });
  }

  var renderer = new THREE.WebGLRenderer({ antialias: true });
  renderer.setPixelRatio(devicePixelRatio || 1);
  renderer.domElement.className = 'gl';
  document.body.prepend(renderer.domElement);
  var scene = new THREE.Scene(); scene.background = new THREE.Color(TH.bg);
  var camera = new THREE.PerspectiveCamera(40, 1, 0.1, 200);
  var controls = new THREE.OrbitControls(camera, renderer.domElement);
  controls.enableDamping = true;
  scene.add(new THREE.AmbientLight(0xffffff, 0.55));
  var key = new THREE.DirectionalLight(0xffffff, 0.6); key.position.set(-6, 10, 12); scene.add(key);

  var plane = { w: 16, h: 0, curvature: 0 }, mesh, tex, baseX;
  function buildSurface() {
    cvs.width = Math.ceil(COLS * CWPX); cvs.height = Math.ceil(ROWS * CH);
    plane.h = plane.w * cvs.height / cvs.width;
    tex = new THREE.CanvasTexture(cvs);
    tex.minFilter = THREE.LinearFilter; tex.anisotropy = renderer.capabilities.getMaxAnisotropy();
    var g = new THREE.PlaneGeometry(plane.w, plane.h, COLS, 1);
    baseX = Float32Array.from(g.attributes.position.array.filter(function (_, n) { return n % 3 === 0; }));
    mesh = new THREE.Mesh(g, new THREE.MeshBasicMaterial({ map: tex, side: THREE.DoubleSide }));
    scene.add(mesh);
  }
  // bend around a vertical cylinder: arc length x maps to angle x/R, edges come toward the viewer
  function bendX(x, c) {
    if (c < 1e-3) return { x: x, z: 0, nx: 0, nz: 1 };
    var R0 = plane.w / (c * Math.PI * 0.9), a = x / R0;
    return { x: R0 * Math.sin(a), z: R0 * (1 - Math.cos(a)), nx: -Math.sin(a), nz: Math.cos(a) };
  }
  function setCurvature(c) {
    plane.curvature = c;
    var pos = mesh.geometry.attributes.position;
    for (var n = 0; n < pos.count; n++) { var b = bendX(baseX[n], c); pos.setX(n, b.x); pos.setZ(n, b.z); }
    pos.needsUpdate = true; mesh.geometry.computeVertexNormals(); mesh.geometry.computeBoundingSphere();
    anchored.forEach(place);
    drawWarp();
  }

  // ---- CellAnchor: centre of cell (row, col) in scene space (fractional cells allowed), plus normal
  function cellToScene(row, col, rows, cols, pl) {
    var b = bendX(((col + 0.5) / cols - 0.5) * pl.w, pl.curvature);
    var p = new THREE.Vector3(b.x, (0.5 - (row + 0.5) / rows) * pl.h, b.z);
    p.normal = new THREE.Vector3(b.nx, 0, b.nz);
    return p;
  }
  function cellSize() { return { w: plane.w / COLS, h: plane.h / ROWS }; }

  // ---- AnchoredObject: {id,row,col,w,h,depth,scale,color,brightness,animate} → group on the surface
  var anchored = [], Z = new THREE.Vector3(0, 0, 1);
  function anchor(spec, obj) {
    var g = new THREE.Group(); g.add(obj); g.userData = spec; spec.group = g; spec.obj = obj;
    scene.add(g); anchored.push(spec); place(spec); return spec;
  }
  function place(a) {
    var p = cellToScene(a.row + (a.h - 1) / 2, a.col + (a.w - 1) / 2, ROWS, COLS, plane);
    a.group.position.copy(p).addScaledVector(p.normal, a.depth || 0);
    a.group.quaternion.setFromUnitVectors(Z, p.normal);
  }

  // ---- LoadColumn: box rising out of the surface, height eases to load/100 × maxH over ~400 ms
  var UNIT = new THREE.BoxGeometry(1, 1, 1).translate(0, 0, 0.5), CAP = new THREE.EdgesGeometry(new THREE.PlaneGeometry(1, 1));
  function loadColumn(spec, maxH) {
    var cs = cellSize(), sc = spec.scale || 1, o = new THREE.Group();
    var mat = new THREE.MeshLambertMaterial({ color: TH[spec.color], emissive: TH[spec.color], emissiveIntensity: 0.25 * (spec.brightness || 1) });
    var body = new THREE.Mesh(UNIT, mat), cap = new THREE.LineSegments(CAP, new THREE.LineBasicMaterial({ color: TH.dim }));
    body.scale.set(spec.w * cs.w * sc, spec.h * cs.h * sc, 0.01); cap.scale.set(body.scale.x, body.scale.y, 1);
    o.add(body, cap);
    spec.cur = 0; spec.target = 0; spec.maxH = maxH; spec.body = body; spec.cap = cap;
    spec.set = function (v, role) { spec.target = Math.max(0, Math.min(1, v)); spec.role = role; mat.color.set(TH[role]); mat.emissive.set(TH[role]); };
    spec.animate = function (t, dt) {
      spec.cur += (spec.target - spec.cur) * (1 - Math.exp(-dt / 0.13));
      var hgt = Math.max(0.01, spec.cur * spec.maxH); body.scale.z = hgt; cap.position.z = hgt + 0.002;
    };
    body.userData.anchor = spec;
    return anchor(spec, o);
  }

  // ---- the die: spins and bobs over the machine name, like Ratty's rat cursor
  function dieBlock() {
    var o = new THREE.Group(), inner = new THREE.Group();
    var body = new THREE.Mesh(new THREE.BoxGeometry(0.42, 0.42, 0.07), new THREE.MeshLambertMaterial({ color: TH.dim, emissive: TH.dim, emissiveIntensity: 0.2 }));
    var edge = new THREE.LineSegments(new THREE.EdgesGeometry(body.geometry), new THREE.LineBasicMaterial({ color: TH.accent }));
    inner.add(body, edge); o.add(inner);
    var spec = { id: 'die', row: 0, col: X, w: M.name.length, h: 1, depth: 0.3, scale: 1, color: 'accent', brightness: 1 };
    spec.animate = function (t) { inner.rotation.y = t * 1.6; inner.position.y = 0.45 + 0.06 * Math.sin(t * 2.4); };
    return anchor(spec, o);
  }

  // ---- TuiHud
  var tipEl = document.getElementById('tip'), statusEl = document.getElementById('status');
  var warpEl = document.getElementById('warp'), legendEl = document.getElementById('legend');
  function hudBox(title, lines) {
    var w = Math.max(title.length + 6, lines.reduce(function (m, l) { return Math.max(m, vis(l)); }, 0) + 4);
    return T.box(title, null, w, 0, lines);
  }
  var speed = 1, rec = window.SAMPLES.samples[0].t.slice(0, 16).replace('T', ' ');
  function drawStatus(i) {
    T.Screen(statusEl, [[['dim', (R.paused() ? 'paused ' : 'replay ') + (i + 1) + '/' + R.count + ' · ' + speed + '× · recorded ' + rec]]]);
  }
  var WARPS = [0, 0.35, 0.7, 1];
  function drawWarp() {
    T.Screen(warpEl, hudBox('warp', [T.bar(plane.curvature * 100, 12, 'text').concat([['', ' ' + plane.curvature.toFixed(2)]]), [['dim', 'w or click · cycle']]]));
  }
  function cycleWarp() { setCurvature(WARPS[(WARPS.indexOf(plane.curvature) + 1) % WARPS.length]); }
  T.Screen(legendEl, hudBox('keys', [
    [['', 'drag'], ['dim', '   orbit   '], ['', 'wheel'], ['dim', '  zoom']], [['', 'click'], ['dim', '  focus a panel']],
    [['', 'space'], ['dim', '  pause   '], ['', '← →'], ['dim', '  step']], [['', '1–4'], ['dim', '    camera presets']],
    [['', 'w'], ['dim', '      warp the slab']], [['', '?'], ['dim', '      this legend']]]));

  // ---- CameraRig: damped orbit, presets 1–4 and fly-to, 600 ms tween (instant if reduced motion)
  var PRESETS = [[-9.5, 4, 17.5], [0, 0, 19], [13, -1.5, 11], [0, 16, 9]], tween = null;
  function fly(pos, target) {
    tween = { p0: camera.position.clone(), t0: controls.target.clone(), p1: pos, t1: target, s: performance.now(), d: reduced ? 0 : 600 };
  }
  function preset(n) { fly(new THREE.Vector3().fromArray(PRESETS[n]), new THREE.Vector3(0, 0, 0)); }
  function focus(reg) {
    var c = cellToScene(reg.row + (reg.h - 1) / 2, reg.col + (reg.w - 1) / 2, ROWS, COLS, plane), cs = cellSize();
    var tn = Math.tan(camera.fov * Math.PI / 360), d = Math.max(reg.h * cs.h / 2 / tn, reg.w * cs.w / 2 / (tn * camera.aspect)) * 1.2;
    fly(c.clone().addScaledVector(c.normal, Math.max(d, 1.5)), c);
  }

  // ---- picking: raycast → uv → cell → region
  var ray = new THREE.Raycaster(), ptr = new THREE.Vector2(), cols = [];
  function pick(e) {
    ptr.set(e.clientX / innerWidth * 2 - 1, -e.clientY / innerHeight * 2 + 1);
    ray.setFromCamera(ptr, camera);
    var hit = ray.intersectObjects(cols.map(function (a) { return a.body; }).concat(mesh))[0];
    if (!hit) return null;
    if (hit.object.userData.anchor) { var a = hit.object.userData.anchor; return { anchor: a, region: a.region }; }
    var row = Math.floor((1 - hit.uv.y) * ROWS), col = Math.floor(hit.uv.x * COLS);
    var reg = REGIONS.filter(function (r) { return row >= r.row && row < r.row + r.h && col >= r.col && col < r.col + r.w; })[0];
    if (!reg) return null;
    var res = { region: reg };
    if (reg.name === 'cores') cols.forEach(function (a) { var c = coreCell(a.core); if (row === c.row && col >= c.col && col < c.col + CORE_CW) res.anchor = a; });
    return res;
  }
  function tipLines(p) {
    var s = R.current(), a = p.anchor;
    if (a && a.core != null) { var v = s.cores[a.core]; return hudBox(coreLabel(a.core), [T.bar(v, 12).concat([[T.level(v), pf(v, 0, 4) + '%']])]); }
    if (a) return hudBox('power', [[['series-power', s.pw.toFixed(1) + ' W', true], ['dim', '  ·  GPU ' + s.pwg.toFixed(1) + ' W']]]);
    return hudBox(p.region.name, [[['dim', 'click to focus']]]);
  }
  var down = null;
  renderer.domElement.addEventListener('pointerdown', function (e) { down = [e.clientX, e.clientY]; });
  renderer.domElement.addEventListener('pointerup', function (e) {
    if (!down || Math.hypot(e.clientX - down[0], e.clientY - down[1]) > 4) return;
    var p = pick(e); if (p) focus(p.region);
  });
  renderer.domElement.addEventListener('pointermove', function (e) {
    var p = pick(e);
    if (!p) { tipEl.style.display = 'none'; return; }
    T.Screen(tipEl, tipLines(p));
    tipEl.style.display = 'block'; tipEl.style.left = (e.clientX + 14) + 'px'; tipEl.style.top = (e.clientY + 14) + 'px';
  });
  renderer.domElement.addEventListener('pointerleave', function () { tipEl.style.display = 'none'; });
  warpEl.addEventListener('click', cycleWarp);

  addEventListener('keydown', function (e) {
    if (e.code === 'Space') { e.preventDefault(); R.toggle(); drawStatus(R.index()); }
    else if ((e.key === 'ArrowLeft' || e.key === 'ArrowRight') && R.paused()) R.step(e.key === 'ArrowLeft' ? -1 : 1);
    else if (e.key >= '1' && e.key <= '4') preset(+e.key - 1);
    else if (e.key === 'w') cycleWarp();
    else if (e.key === '?') legendEl.style.display = legendEl.style.display === 'block' ? 'none' : 'block';
  });
  function resize() { renderer.setSize(innerWidth, innerHeight, false); camera.aspect = innerWidth / innerHeight; camera.updateProjectionMatrix(); }
  addEventListener('resize', resize);

  // ---- start once the face is loaded, so the first canvas draw is in JetBrains Mono
  Promise.all([document.fonts.load('400 11.5px "JetBrains Mono"'), document.fonts.load('700 11.5px "JetBrains Mono"')])
    .then(function () { return document.fonts.ready; })
    .then(function () {
      ctx.font = '400 ' + FPX + 'px ' + FONT; CWPX = ctx.measureText('M').width;
      buildSurface(); resize();
      camera.position.fromArray(PRESETS[0]); controls.target.set(0, 0, 0);
      var coresReg = REGIONS.filter(function (r) { return r.name === 'cores'; })[0];
      for (var k = 0; k < 10; k++) {
        // stand on the last 4 cells of the core's bar (mostly off-cells), not on its % readout
        var c = coreCell(k), a = loadColumn({ id: 'core' + k, row: c.row, col: c.col + 3 + CORE_BW - 4, w: 4, h: 1, depth: 0, scale: 0.8, color: 'level-low', brightness: 1 }, 1.5);
        a.core = k; a.region = coresReg; cols.push(a);
      }
      var pwCol = loadColumn({ id: 'power', row: 2, col: X + 2 * (TW + 1) + 6, w: 4, h: 1, depth: 0, scale: 0.8, color: 'series-power', brightness: 1 }, 1.2);
      pwCol.region = REGIONS[3]; cols.push(pwCol);
      dieBlock();
      setCurvature(0.35);
      R.onSample(function (s, h, i) {
        drawScreen(compose(s, h)); tex.needsUpdate = true;
        cols.forEach(function (a) { if (a.core != null) a.set(s.cores[a.core] / 100, T.level(s.cores[a.core])); });
        pwCol.set(s.pw / hmax(h.pw, 1), 'series-power');
        drawStatus(i);
      });
      var last = performance.now();
      (function frame(now) {
        var dt = Math.min(0.1, (now - last) / 1000); last = now;
        if (tween) {
          var k = tween.d ? Math.min(1, (now - tween.s) / tween.d) : 1, e = k < 0.5 ? 4 * k * k * k : 1 - Math.pow(-2 * k + 2, 3) / 2;
          camera.position.lerpVectors(tween.p0, tween.p1, e); controls.target.lerpVectors(tween.t0, tween.t1, e);
          if (k >= 1) tween = null;
        }
        controls.update();
        anchored.forEach(function (a) { if (a.animate) a.animate(now / 1000, dt); });
        renderer.render(scene, camera);
        requestAnimationFrame(frame);
      })(last);
    });
})();
