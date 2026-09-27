/* @ds-bundle: {"format":4,"namespace":"MonitorTUI","components":[{"name":"Box"},{"name":"Tile"},{"name":"Bar"},{"name":"Sparkline"},{"name":"BrailleGraph"},{"name":"Header"},{"name":"ProcessTable"},{"name":"RunBars"},{"name":"SectionedFrame"},{"name":"StatRow"}]} */
// A line-for-line port of bin/monitor's and drafts/'s drawing functions. A line is an array of
// segments [role, text, bold]; role is a colour token name ('' = text).
window.MonitorTUI = (function () {
  function chars(s) { return Array.from(s); }
  function rep(c, n) { return n > 0 ? c.repeat(n) : ''; }
  function vis(line) { return line.reduce(function (n, s) { return n + chars(s[1]).length; }, 0); }
  function plain(line) { return line.map(function (s) { return s[1]; }).join(''); }

  function level(p) { return p >= 80 ? 'level-high' : p >= 50 ? 'level-mid' : 'level-low'; }

  function fit(s, w) {
    var a = chars(s);
    return a.length > w ? a.slice(0, w - 1).join('') + '…' : s + rep(' ', w - a.length);
  }

  function padv(line, w) {
    var n = vis(line);
    if (n > w) return [['', fit(plain(line), w)]];
    return line.concat([['', rep(' ', w - n)]]);
  }

  function spread(l, r, w) {
    var n = w - vis(l) - vis(r);
    return l.concat([['', rep(' ', n < 1 ? 1 : n)]], r);
  }

  function center(line, w) {
    var l = Math.floor((w - vis(line)) / 2);
    return [['', rep(' ', l < 0 ? 0 : l)]].concat(line);
  }

  // percent, width, [role], [on], [off]
  function bar(p, w, role, on, off) {
    var q = Math.min(p, 100), k = Math.floor(q / 100 * w + 0.5);
    return [[role || level(p), rep(on || '■', k)], ['dim', rep(off || '·', w - k)]];
  }

  var SPK = chars('▁▂▃▄▅▆▇█');
  // values (newest last), width, max -> one segment string, blank where there is no data
  function spark(vals, w, max) {
    var s = '', n = vals.length, i, x;
    if (!(max > 0)) max = 1;
    for (i = n - w + 1; i <= n; i++) {
      if (i < 1) { s += ' '; continue; }
      x = Math.min(Math.floor(vals[i - 1] / max * 7 + 0.5), 7);
      s += SPK[x];
    }
    return s;
  }

  var FILL_L = [0, 64, 68, 70, 71], FILL_R = [0, 128, 160, 176, 184];
  // values, w, h, max, [fixed role] -> lines; two samples per cell, four dots per row
  function graph(vals, w, h, max, role) {
    var n = vals.length, cells = new Array(w * h).fill(0), c, side, idx, v, dots, r, k, out = [];
    if (!(max > 0)) max = 1;
    for (c = 1; c <= w; c++) for (side = 1; side <= 2; side++) {
      idx = n - 2 * w + (c - 1) * 2 + side;
      if (idx < 1) continue;
      v = vals[idx - 1];
      dots = Math.floor(Math.min(v / max, 1) * h * 4 + 0.5);
      if (v > 0 && dots === 0) dots = 1;
      for (r = h; r >= 1 && dots > 0; r--) {
        k = Math.min(dots, 4);
        cells[(r - 1) * w + c - 1] += (side === 1 ? FILL_L : FILL_R)[k];
        dots -= k;
      }
    }
    for (r = 1; r <= h; r++) {
      var row = '';
      for (c = 1; c <= w; c++) row += String.fromCharCode(0x2800 + cells[(r - 1) * w + c - 1]);
      out.push([[role || level((h - r + 0.5) / h * 100), row]]);
    }
    return out;
  }

  // frame_chars in drafts/core.zsh: corners TL TR BL BR, edge, side. heavy is bin/monitor's.
  var FRAMES = {
    heavy: ['┏', '┓', '┗', '┛', '━', '┃'],
    round: ['╭', '╮', '╰', '╯', '─', '│'],
    square: ['┌', '┐', '└', '┘', '─', '│'],
    double: ['╔', '╗', '╚', '╝', '═', '║'],
    dashed: ['┌', '┐', '└', '┘', '┄', '┆'],
    rule: ['─', '─', ' ', ' ', '─', ' ']
  };

  // title, right label (line or null), width, height (0 = fit), lines, [frame]
  function box(t, right, w, h, lines, frame) {
    var B = FRAMES[frame || 'heavy'], L = lines.slice(), iw = w - 4, out = [], tl = ' ' + t + ' ';
    var tr = right && right.length ? [['', ' ']].concat(right, [['', ' ']]) : [];
    while (h > 0 && L.length < h - 2) L.push([]);
    if (h > 0 && L.length > h - 2) L = L.slice(0, h - 2);
    var n = Math.max(w - 4 - chars(tl).length - vis(tr), 0);
    out.push([['dim', B[0] + B[4]], ['', tl, true], ['dim', rep(B[4], n)]].concat(tr, [['dim', B[4] + B[1]]]));
    L.forEach(function (l) { out.push([['dim', B[5]], ['', ' ']].concat(padv(l, iw), [['', ' '], ['dim', B[5]]])); });
    out.push(frame === 'rule' ? [['', rep(' ', w)]] : [['dim', B[2] + rep(B[4], w - 2) + B[3]]]);
    return out;
  }

  // title, value line, values, max, role, width -> a 4-row KPI tile
  function tile(t, value, vals, max, role, w) {
    var iw = w - 4, s = vals ? [[role || '', spark(vals, iw, max)]] : [];
    return box(t, null, w, 4, [center(value, iw), s]);
  }

  // gap, blocks... -> lines side by side, each block padded to its widest line
  function hjoin(gap) {
    var blocks = Array.prototype.slice.call(arguments, 1), h = 0, ws = [], out = [], i, j;
    blocks.forEach(function (b) {
      h = Math.max(h, b.length);
      ws.push(b.reduce(function (m, l) { return Math.max(m, vis(l)); }, 0));
    });
    for (i = 0; i < h; i++) {
      var line = [];
      for (j = 0; j < blocks.length; j++) {
        line = line.concat(padv(blocks[j][i] || [], ws[j]));
        if (j < blocks.length - 1) line.push(['', rep(' ', gap)]);
      }
      out.push(line);
    }
    return out;
  }

  var VB = [' ', '▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'];
  // vbar2: series a, series b, height, max, bar width, role a, role b -> lines; one a|b pair
  // per sample with a space after each pair, eight steps per row, newest on the right
  function runBars(A, Bs, h, max, bw, ra, rb) {
    var out = [], r, i, e;
    if (!(max > 0)) max = 1;
    function step(v) { e = Math.floor(v / max * h * 8 + 0.5) - (r - 1) * 8; return VB[Math.max(0, Math.min(8, e))]; }
    for (r = h; r >= 1; r--) {
      var line = [];
      for (i = 0; i < A.length; i++) line.push([ra, rep(step(A[i]), bw)], [rb, rep(step(Bs[i]), bw)], ['', ' ']);
      out.push(line);
    }
    return out;
  }

  // v04's instrument panel: one outer frame split into two columns by shared ├ ┼ ┤ dividers.
  // lw, rw = column widths; bands = [{left: [title, right, lines], right: [title, right, lines]}];
  // bottom = [title, lines] spans the full width. Total width is lw + rw + 3.
  function seg(t, right, w) {
    var tl = ' ' + t + ' ', tr = right && right.length ? [['', ' ']].concat(right, [['', ' ']]) : [];
    var n = Math.max(w - 2 - chars(tl).length - vis(tr), 0);
    return [['dim', '─'], ['', tl, true], ['dim', rep('─', n)]].concat(tr, [['dim', '─']]);
  }
  function sectioned(lw, rw, bands, bottom) {
    var out = [], w = lw + rw + 3;
    bands.forEach(function (b, k) {
      var j = k ? ['├', '┼', '┤'] : ['╭', '┬', '╮'], Lc = b.left[2], Rc = b.right[2], i;
      out.push([['dim', j[0]]].concat(seg(b.left[0], b.left[1], lw), [['dim', j[1]]], seg(b.right[0], b.right[1], rw), [['dim', j[2]]]));
      for (i = 0; i < Math.max(Lc.length, Rc.length); i++)
        out.push([['dim', '│'], ['', ' ']].concat(padv(Lc[i] || [], lw - 2), [['', ' '], ['dim', '│'], ['', ' ']], padv(Rc[i] || [], rw - 2), [['', ' '], ['dim', '│']]));
    });
    out.push([['dim', '├']].concat(seg(bottom[0], null, lw), [['dim', '┴' + rep('─', rw) + '┤']]));
    bottom[1].forEach(function (l) { out.push([['dim', '│'], ['', ' ']].concat(padv(l, w - 4), [['', ' '], ['dim', '│']])); });
    out.push([['dim', '╰' + rep('─', w - 2) + '╯']]);
    return out;
  }

  // v06's sidebar row: dim label padded to labelW, the value, and optionally a ▰▱ bar
  // right-aligned in the last iw − 21 cells
  function statRow(label, value, pct, iw, labelW) {
    var l = [['dim', fit(label, labelW || 7)]].concat(value);
    if (pct == null) return l;
    return padv(l, 21).concat(bar(pct, iw - 21, '', '▰', '▱'));
  }
  // a dim subheading rule inside a column: ── title ─────
  function rule(t, w) { return [['dim', '── ' + t + ' ' + rep('─', w - chars(t).length - 4)]]; }

  // header line: machine facts left, battery and clock (or starting note) right
  function header(m, w) {
    var l = [['', m.name, true], ['dim', '  ·  ' + m.e + 'E + ' + m.p + 'P CPU  ·  ' + m.gpuCores + '-core GPU']];
    var r = m.have ? [['dim', (m.battery ? 'battery ' + m.battery + '   ' : '') + m.clock]] : [['level-mid', '● starting mactop…']];
    return [spread(l, r, w)];
  }

  function pf(n, d, w) { var s = n.toFixed(d); return rep(' ', w - s.length) + s; }

  // rows, width, count -> the process table; columns drop out below 72 and 48 cells
  function processTable(rows, w, count) {
    var cols = w < 48 ? 'min' : w < 72 ? 'mid' : 'full';
    var nw = cols === 'full' ? w - 50 : cols === 'mid' ? w - 23 : w - 8, out = [];
    var head = cols === 'full' ? ' PID     ' + fit('COMMAND', nw) + '   CPU%   GPU ms/s    MEM%        RSS'
      : cols === 'mid' ? ' PID     ' + fit('COMMAND', nw) + '   CPU%    MEM%' : ' ' + fit('COMMAND', nw) + '   CPU%';
    out.push([['', head, true]]);
    for (var i = 0; i < (count || rows.length); i++) {
      var p = rows[i];
      if (!p) { out.push([]); continue; }
      var lv = level(p.cpu), a = fit(String(p.pid), 8), b = fit(p.command, nw);
      if (cols === 'full') out.push([['dim', ' ' + a], ['', b], [lv, pf(p.cpu, 1, 7)], ['series-gpu', pf(p.gpu, 1, 11)], ['dim', pf(p.mem, 1, 8) + pf(p.rss / 1024, 0, 8) + ' MB']]);
      else if (cols === 'mid') out.push([['dim', ' ' + a], ['', b], [lv, pf(p.cpu, 1, 7)], ['dim', pf(p.mem, 1, 8)]]);
      else out.push([['', ' ' + b], [lv, pf(p.cpu, 1, 7)]]);
    }
    return out;
  }

  function esc(s) { return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;'); }
  // lines -> HTML for a <pre class="mt-screen">
  function html(lines) {
    return lines.map(function (line) {
      return line.map(function (s) {
        var cls = (s[0] ? 'mt-' + s[0] : '') + (s[2] ? ' mt-b' : '');
        return cls ? '<span class="' + cls.trim() + '">' + esc(s[1]) + '</span>' : esc(s[1]);
      }).join('');
    }).join('\n');
  }

  function Screen(el, lines) { el.classList.add('mt-screen'); el.innerHTML = html(lines); return el; }

  return {
    level: level, fit: fit, bar: bar, spark: spark, graph: graph, box: box, tile: tile,
    hjoin: hjoin, center: center, spread: spread, header: header, processTable: processTable,
    runBars: runBars, sectioned: sectioned, statRow: statRow, rule: rule, FRAMES: FRAMES,
    html: html, Screen: Screen,
    Box: box, Tile: tile, Bar: bar, Sparkline: spark, BrailleGraph: graph, Header: header, ProcessTable: processTable,
    RunBars: runBars, SectionedFrame: sectioned, StatRow: statRow
  };
})();
