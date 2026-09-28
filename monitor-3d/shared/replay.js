// Replays the 90 recorded mactop samples (render/mactop.raw) at 1 Hz, looping, with a
// 400-sample history per series like bin/monitor. Needs samples.js loaded first.
window.Replay = (function () {
  var S = window.SAMPLES.samples, meta = window.SAMPLES.meta, HIST = 400;
  var i = 0, paused = false, speed = 1, timer = null, subs = [];
  var hist = { cpu: [], gpu: [], pw: [], mem: [], tc: [], nin: [], cores: [] };
  function cur() { return S[i]; }
  function push(s) {
    hist.cpu.push(s.cpu); hist.gpu.push(s.gpu); hist.pw.push(s.pw);
    hist.mem.push(s.mu * 100 / s.mt); hist.tc.push(s.tc); hist.nin.push(s.nin); hist.cores.push(s.cores);
    for (var k in hist) if (hist[k].length > HIST) hist[k].shift();
  }
  function step(d) {
    i = (i + (d || 1) + S.length) % S.length;
    if (d < 0) { for (var k in hist) hist[k].pop(); } else push(S[i]); // back = drop the newest
    subs.forEach(function (f) { f(S[i], hist, i); });
  }
  function loop() { clearTimeout(timer); timer = setTimeout(function () { if (!paused) step(1); loop(); }, 1000 / speed); }
  // prefill history with every recorded sample so graphs and the terrain start full
  for (var k = 0; k < S.length; k++) { i = k; push(S[k]); }
  loop();
  return {
    meta: meta, count: S.length, hist: hist, current: cur, index: function () { return i; },
    onSample: function (f) { subs.push(f); f(S[i], hist, i); },
    step: step,
    toggle: function () { paused = !paused; return paused; },
    paused: function () { return paused; },
    setSpeed: function (x) { speed = x; loop(); }
  };
})();
