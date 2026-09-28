// Shared scene, CameraRig, TuiHud and keys for prototypes B and C.
// opts: {presets: [[pos, target] x4], legend: [[key, desc]], title: line, onArrow(d), onKey(k), onSample(s, hist, i)}
window.BC = function (opts) {
  var T = THREE, M = MonitorTUI, R = Replay;
  var col = function (tok) { return new T.Color(THEME[tok]); };
  var lvl = function (p) { return THEME[M.level(p)]; };

  var renderer = new T.WebGLRenderer({ antialias: true });
  renderer.setPixelRatio(devicePixelRatio);
  renderer.setClearColor(THEME.bg);
  document.body.appendChild(renderer.domElement);
  var scene = new T.Scene();
  var camera = new T.PerspectiveCamera(40, 1, 0.1, 500);
  scene.add(new T.AmbientLight(0xffffff, 0.5));
  var key = new T.DirectionalLight(0xffffff, 0.4);
  key.position.set(6, 12, 8);
  scene.add(key);
  var controls = new T.OrbitControls(camera, renderer.domElement);
  controls.enableDamping = true;

  function resize() {
    renderer.setSize(innerWidth, innerHeight);
    camera.aspect = innerWidth / innerHeight;
    camera.updateProjectionMatrix();
  }
  addEventListener('resize', resize);
  resize();

  // TuiHud: every overlay is a MonitorTUI screen
  function pre(id, cls) {
    var el = document.createElement('pre');
    if (id) el.id = id;
    el.className = 'hud ' + (cls || '');
    document.body.appendChild(el);
    return el;
  }
  function vis(l) { return l.reduce(function (n, s) { return n + Array.from(s[1]).length; }, 0); }
  function boxed(title, lines) {
    var w = Math.max(Array.from(title).length + 6, lines.reduce(function (m, l) { return Math.max(m, vis(l)); }, 0) + 4);
    return M.box(title, null, w, 0, lines);
  }
  var tipEl = pre('tip'), statusEl = pre('status'), legendEl = pre('legend'), titleEl = pre('title');
  M.Screen(titleEl, [opts.title]);
  var keys = [['drag', 'orbit'], ['wheel', 'zoom'], ['hover', 'tooltip'], ['click', 'isolate'], ['space', 'pause'],
    ['← →', 'step while paused'], ['1-4', 'camera presets']].concat(opts.legend || [], [['?', 'this legend']]);
  M.Screen(legendEl, boxed('keys', keys.map(function (k) { return [['', M.fit(k[0], 7), true], ['dim', k[1]]]; })));
  legendEl.style.display = 'none';

  function tip(e, title, lines) {
    if (!title) { tipEl.style.display = 'none'; return; }
    M.Screen(tipEl, boxed(title, lines));
    tipEl.style.display = 'block';
    var r = tipEl.getBoundingClientRect(), x = e.clientX + 14, y = e.clientY + 14;
    tipEl.style.left = (x + r.width > innerWidth ? e.clientX - 14 - r.width : x) + 'px';
    tipEl.style.top = (y + r.height > innerHeight ? e.clientY - 14 - r.height : y) + 'px';
  }

  var t0 = SAMPLES.samples[0].t, rec = t0.slice(0, 10) + ' ' + t0.slice(11, 16);
  function status() {
    M.Screen(statusEl, [[['dim', 'replay ' + (R.index() + 1) + '/' + R.count + ' · ' + (R.paused() ? 'paused' : '1×') + ' · recorded ' + rec]]]);
  }

  // CameraRig: presets on 1-4, 600 ms tween, cut under prefers-reduced-motion
  var reduce = matchMedia('(prefers-reduced-motion: reduce)').matches, tw = null;
  function preset(n) {
    var p = opts.presets[n], to = new T.Vector3().fromArray(p[0]), tt = new T.Vector3().fromArray(p[1]);
    if (reduce) { camera.position.copy(to); controls.target.copy(tt); tw = null; return; }
    tw = { t: performance.now(), p0: camera.position.clone(), t0: controls.target.clone(), p1: to, t1: tt };
  }
  camera.position.fromArray(opts.presets[0][0]);
  controls.target.fromArray(opts.presets[0][1]);

  // projected DOM labels
  var labels = [];
  function label(pos, lines) {
    var el = pre(null, 'plain lbl');
    M.Screen(el, lines);
    var o = { el: el, pos: pos };
    labels.push(o);
    return o;
  }

  // picking: hover every move, click only when the pointer didn't drag
  var ray = new T.Raycaster(), ndc = new T.Vector2(), down = null;
  function pick(e, objs) {
    ndc.set(e.clientX / innerWidth * 2 - 1, -e.clientY / innerHeight * 2 + 1);
    ray.setFromCamera(ndc, camera);
    return ray.intersectObjects(objs, false)[0] || null;
  }
  var cv = renderer.domElement;
  cv.addEventListener('pointermove', function (e) { if (opts.onHover) opts.onHover(e); });
  cv.addEventListener('pointerleave', function () { tip(null); });
  cv.addEventListener('pointerdown', function (e) { down = [e.clientX, e.clientY]; });
  cv.addEventListener('pointerup', function (e) {
    if (down && Math.hypot(e.clientX - down[0], e.clientY - down[1]) < 4 && opts.onClick) opts.onClick(e);
    down = null;
  });

  addEventListener('keydown', function (e) {
    if (e.key === ' ') { R.toggle(); status(); e.preventDefault(); }
    else if (e.key === 'ArrowLeft' || e.key === 'ArrowRight') {
      var d = e.key === 'ArrowLeft' ? -1 : 1;
      if (R.paused()) { if (opts.onArrow) opts.onArrow(d); else R.step(d); }
    }
    else if (e.key >= '1' && e.key <= '4') preset(+e.key - 1);
    else if (e.key === '?') legendEl.style.display = legendEl.style.display === 'none' ? 'block' : 'none';
    else if (opts.onKey) opts.onKey(e.key);
  });

  var frames = [], last = performance.now(), v = new T.Vector3();
  function loop(now) {
    var dt = Math.min((now - last) / 1000, 0.1);
    last = now;
    if (tw) {
      var k = Math.min((now - tw.t) / 600, 1), s = k < 0.5 ? 2 * k * k : 1 - Math.pow(-2 * k + 2, 2) / 2;
      camera.position.lerpVectors(tw.p0, tw.p1, s);
      controls.target.lerpVectors(tw.t0, tw.t1, s);
      if (k === 1) tw = null;
    }
    controls.update();
    frames.forEach(function (f) { f(dt); });
    renderer.render(scene, camera);
    labels.forEach(function (l) {
      v.copy(l.pos).project(camera);
      l.el.style.display = v.z < 1 && l.el.dataset.hide !== '1' ? 'block' : 'none';
      l.el.style.left = ((v.x + 1) / 2 * innerWidth) + 'px';
      l.el.style.top = ((1 - v.y) / 2 * innerHeight) + 'px';
    });
    requestAnimationFrame(loop);
  }
  requestAnimationFrame(loop);

  return { T: T, M: M, scene: scene, camera: camera, col: col, lvl: lvl, tip: tip, label: label, pick: pick,
    frame: function (f) { frames.push(f); }, pre: pre,
    start: function () { R.onSample(function (s, h, i) { if (opts.onSample) opts.onSample(s, h, i); status(); }); } };
};
