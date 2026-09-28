#!/usr/bin/env python3
"""monitor3d — cores as a rotating 3D bar chart in braille, inside a Monitor TUI heavy frame.
Replays ../../../shared/samples.json at 1 Hz; --live reads mactop. In Ratty, hands the columns to RGP."""
import json, math, os, select, shutil, signal, subprocess, sys, termios, threading, time, tty

HERE = os.path.dirname(os.path.abspath(__file__))
SAMPLES = os.path.join(HERE, '..', '..', '..', 'shared', 'samples.json')
CUBE = os.path.join(HERE, 'cube.obj')

E = '\x1b'
R, BOLD, DIM = E + '[0m', E + '[1m', E + '[90m'
LOW, MID, HIGH = E + '[32m', E + '[33m', E + '[31m'
HEX = {LOW: 'a3be8c', MID: 'c898ca', HIGH: 'f06459'}  # Alacritty palette: green, yellow (renders lilac), red
FPS, MIN_W, MIN_H = 20, 60, 20
HMAX, D, S = 3.0, 10.0, 0.3  # column max height, camera distance, column half-width (world units)
BITS = ((0x01, 0x08), (0x02, 0x10), (0x04, 0x20), (0x40, 0x80))  # braille [dy][dx]


def level(v):
    return HIGH if v >= 80 else MID if v >= 50 else LOW


def vis(s):
    n, i = 0, 0
    while i < len(s):
        if s[i] == E:
            i = s.index('m', i) + 1
        else:
            n, i = n + 1, i + 1
    return n


# ---------------------------------------------------------------------------
# data

class Replay:
    def __init__(self):
        d = json.load(open(SAMPLES))
        self.meta, self.samples, self.i = d['meta'], d['samples'], -1

    def next(self):
        self.i = (self.i + 1) % len(self.samples)
        return self.samples[self.i]


class Live:
    def __init__(self):
        def sysctl(k, dflt):
            try:
                return subprocess.run(['sysctl', '-n', k], capture_output=True, text=True).stdout.strip() or dflt
            except OSError:
                return dflt
        self.meta = {'name': sysctl('machdep.cpu.brand_string', '?'), 'e': sysctl('hw.perflevel1.physicalcpu', '?'),
                     'p': sysctl('hw.perflevel0.physicalcpu', '?'), 'gpuCores': '?'}
        self.latest, self.seen = None, 0
        self.proc = subprocess.Popen(['mactop', '--headless', '--count', '0', '-i', '1000'], stdin=subprocess.DEVNULL,
                                     stdout=subprocess.PIPE, stderr=subprocess.DEVNULL, text=True)
        threading.Thread(target=self.read, daemon=True).start()

    def read(self):
        for line in self.proc.stdout:
            line = line.strip().lstrip('[,')
            if not line.startswith('{'):
                continue
            try:
                j = json.loads(line.rstrip(']'))
            except ValueError:
                continue
            self.meta['gpuCores'] = j.get('system_info', {}).get('gpu_core_count', '?')
            self.latest = {'t': time.strftime('%H:%M:%S'), 'cpu': j['cpu_usage'], 'cores': j['core_usages']}

    def next(self):
        if self.latest is None or self.seen == id(self.latest):
            return None
        self.seen = id(self.latest)
        return self.latest

    def close(self):
        self.proc.terminate()


# ---------------------------------------------------------------------------
# scene: E row behind, P row in front, one grid square per core

def core_layout(ne, np_):
    return ([((i - (ne - 1) / 2), -0.6, 'E%d' % (i + 1)) for i in range(ne)] +
            [((i - (np_ - 1) / 2), 0.6, 'P%d' % (i + 1)) for i in range(np_)])


class Camera:
    def __init__(self, yaw, pitch):
        self.yaw, self.pitch = yaw, pitch
        self.cy, self.sy_, self.cp, self.sp = math.cos(yaw), math.sin(yaw), math.cos(pitch), math.sin(pitch)
        c = HMAX * 0.4
        self.center = c
        self.pos = (D * self.cp * self.sy_, c + D * self.sp, D * self.cp * self.cy)

    def view(self, x, y, z):  # -> screen x, screen y (up), distance
        y -= self.center
        x1 = x * self.cy - z * self.sy_
        z1 = x * self.sy_ + z * self.cy
        y2 = y * self.cp - z1 * self.sp
        z2 = y * self.sp + z1 * self.cp
        d = D - z2
        return x1 / d, y2 / d, d


def column_corners(x, z):
    return [(x - S, z - S), (x + S, z - S), (x + S, z + S), (x - S, z + S)]


def label_point(cam, x, z):
    return x + 0.62 * math.sin(cam.yaw), 0.0, z + 0.62 * math.cos(cam.yaw)


FIT = {}


def fit(pitch, cores, dw, dh):
    """Dot-space scale/offset fitted over every yaw, so auto-spin doesn't breathe."""
    key = (round(pitch, 4), dw, dh)
    if key not in FIT:
        xs, ys = [], []
        for k in range(24):
            cam = Camera(k * math.pi / 12, pitch)
            for x, z, _ in cores:
                pts = [(a, y, b) for a, b in column_corners(x, z) for y in (0.0, HMAX)] + [label_point(cam, x, z)]
                for p in pts:
                    sx, sy, _ = cam.view(*p)
                    xs.append(sx)
                    ys.append(sy)
        k = min((dw - 4) / (max(xs) - min(xs)), (dh - 6) / (max(ys) - min(ys)))
        FIT[key] = (k, dw / 2 - k * (max(xs) + min(xs)) / 2, dh / 2 + k * (max(ys) + min(ys)) / 2)
    return FIT[key]


class Canvas:
    def __init__(self, cw, ch):
        self.cw, self.ch, self.dw, self.dh = cw, ch, cw * 2, ch * 4
        n = self.dw * self.dh
        self.on, self.owner, self.floor = bytearray(n), [-1] * n, bytearray(n)

    def put(self, x, y, v, owner):
        if 0 <= x < self.dw and 0 <= y < self.dh:
            i = y * self.dw + x
            if owner < 0:
                self.floor[i] = 1
            else:
                self.on[i], self.owner[i] = v, owner

    def line(self, a, b, owner):
        (x0, y0), (x1, y1) = (int(round(a[0])), int(round(a[1]))), (int(round(b[0])), int(round(b[1])))
        dx, dy = abs(x1 - x0), -abs(y1 - y0)
        sx, sy, err = (1 if x0 < x1 else -1), (1 if y0 < y1 else -1), dx + dy
        while True:
            self.put(x0, y0, 1, owner)
            if x0 == x1 and y0 == y1:
                return
            e2 = 2 * err
            if e2 >= dy:
                err, x0 = err + dy, x0 + sx
            if e2 <= dx:
                err, y0 = err + dx, y0 + sy

    def poly(self, pts, pattern, owner):
        ys = [p[1] for p in pts]
        for y in range(max(0, math.ceil(min(ys) - 0.5)), min(self.dh - 1, math.floor(max(ys) - 0.5)) + 1):
            yc, xs = y + 0.5, []
            for (ax, ay), (bx, by) in zip(pts, pts[1:] + pts[:1]):
                if (ay <= yc < by) or (by <= yc < ay):
                    xs.append(ax + (yc - ay) * (bx - ax) / (by - ay))
            if len(xs) < 2:
                continue
            for x in range(max(0, math.ceil(min(xs) - 0.5)), min(self.dw - 1, math.floor(max(xs) - 0.5)) + 1):
                self.put(x, y, pattern(x, y), owner)

    def cells(self, colours):
        """-> rows of (char, colour or None); colour = frontmost (highest draw rank) owner in the cell."""
        out, dw = [], self.dw
        for cy in range(self.ch):
            row = []
            for cx in range(self.cw):
                bits = fbits = 0
                best = -1
                for dy in range(4):
                    base = (cy * 4 + dy) * dw + cx * 2
                    for dx in range(2):
                        i = base + dx
                        o = self.owner[i]
                        if o >= 0:
                            if self.on[i]:
                                bits |= BITS[dy][dx]
                            if o > best:
                                best = o
                        elif self.floor[i]:
                            fbits |= BITS[dy][dx]
                if best >= 0:
                    row.append((chr(0x2800 + bits) if bits else ' ', colours[best]))
                elif fbits:
                    row.append((chr(0x2800 + fbits), DIM))
                else:
                    row.append((' ', None))
            out.append(row)
        return out


FULL = lambda x, y: 1
STRIPE = lambda x, y: x % 2 == 0
CHECK = lambda x, y: (x + y) % 2 == 0


def render_chart(cw, ch, cam, cores, loads, columns):
    """Braille chart -> (cell rows, label anchors [(cell_row, cell_col)], scale ticks [(row, col, text)])."""
    cv = Canvas(cw, ch)
    k, ox, oy = fit(cam.pitch, cores, cv.dw, cv.dh)

    def proj(x, y, z):
        sx, sy, _ = cam.view(x, y, z)
        return ox + sx * k, oy - sy * k

    # floor grid in dim: one square per core slot
    for gx in range(-3, 4):
        cv.line(proj(gx, 0, -1.2), proj(gx, 0, 1.2), -1)
    for gz in (-1.2, 0.0, 1.2):
        cv.line(proj(-3, 0, gz), proj(3, 0, gz), -1)

    # scale post in dim at the back-left corner: 0, 50 and 100% of HMAX, so headroom reads as scale
    ticks = []
    post = proj(-3, 0, -1.2)
    cv.line(post, proj(-3, HMAX, -1.2), -1)
    for f, text in ((0.5, '50%'), (1.0, '100%')):
        a, b = proj(-3, HMAX * f, -1.2), proj(-2.7, HMAX * f, -1.2)
        cv.line(a, b, -1)
        ticks.append((int(a[1] // 4), int(round(a[0] / 2)) - len(text) - 1, text))

    colours, anchors = [], []
    for x, z, _ in cores:
        lx, ly = proj(*label_point(cam, x, z))
        anchors.append((int(ly // 4), int(round(lx / 2)) - 1))
    if columns:
        order = sorted(range(len(cores)), key=lambda i: -cam.view(cores[i][0], 0, cores[i][1])[2])  # far first
        lx_, lz_ = -0.55, 0.83  # side light, world x/z
        for rank, i in enumerate(order):
            x, z, _ = cores[i]
            h = max(0.04, loads[i] / 100 * HMAX)
            c = column_corners(x, z)
            bot = [proj(a, 0, b) for a, b in c]
            top = [proj(a, h, b) for a, b in c]
            faces = [((0, 1, 0), top, (x, h, z))]
            for (a, b), n in (((3, 2), (0, 0, 1)), ((0, 1), (0, 0, -1)), ((1, 2), (1, 0, 0)), ((0, 3), (-1, 0, 0))):
                fc = (x + n[0] * S, h / 2, z + n[2] * S)
                faces.append((n, [bot[a], bot[b], top[b], top[a]], fc))
            colours.append(level(loads[i]))
            for n, pts, fc in faces:
                to_cam = [cam.pos[j] - fc[j] for j in range(3)]
                if sum(n[j] * to_cam[j] for j in range(3)) <= 0:
                    continue  # back face
                pat = FULL if n[1] else (STRIPE if n[0] * lx_ + n[2] * lz_ > 0 else CHECK)
                cv.poly(pts, pat, rank)
                for p, q in zip(pts, pts[1:] + pts[:1]):
                    cv.line(p, q, rank)
    return cv.cells(colours), anchors, ticks


# ---------------------------------------------------------------------------
# ratty graphics protocol (APC: ESC _ ratty;g;<verb>;k=v... ESC \)

def rgp(verb, **kv):
    return E + '_ratty;g;' + ';'.join([verb] + ['%s=%s' % kv_ for kv_ in kv.items()]) + E + '\\'


def detect_ratty(fd):
    os.write(1, rgp('s').encode())
    buf, end = b'', time.monotonic() + 0.15
    while time.monotonic() < end:
        r, _, _ = select.select([fd], [], [], max(0, end - time.monotonic()))
        if not r:
            break
        buf += os.read(fd, 256)
        if E.encode() + b'\\' in buf:
            break
    return (E + '_ratty;g;s;v=').encode() in buf


# ---------------------------------------------------------------------------
# frame

def box(title, right, w, lines):
    tl, tr = ' %s ' % title, ' %s ' % right
    n = max(0, w - 4 - vis(tl) - vis(tr))
    out = [DIM + '┏━' + R + BOLD + tl + R + DIM + '━' * n + R + tr + DIM + '━┓' + R]
    for l in lines:
        out.append(DIM + '┃' + R + ' ' + l + ' ' * max(0, w - 4 - vis(l)) + ' ' + DIM + '┃' + R)
    out.append(DIM + '┗' + '━' * (w - 2) + '┛' + R)
    return out


def spread(left, right, w):
    return left + ' ' * max(1, w - vis(left) - vis(right)) + right


def main():
    args = sys.argv[1:]
    frames = int(args[args.index('--frames') + 1]) if '--frames' in args else 0
    src = None
    if '--live' in args and shutil.which('mactop'):
        src = Live()
    if src is None:
        src = Replay()
    meta = src.meta
    ne, np_ = int(meta['e']) if str(meta['e']).isdigit() else 4, int(meta['p']) if str(meta['p']).isdigit() else 6
    cores = core_layout(ne, np_)
    n = len(cores)

    fd = sys.stdin.fileno()
    is_tty = os.isatty(fd)
    saved = termios.tcgetattr(fd) if is_tty else None
    resized = [True]
    signal.signal(signal.SIGWINCH, lambda *_: resized.__setitem__(0, True))
    out = sys.stdout
    placed = {}
    ratty_ok = ratty = '--force-ratty' in args
    try:
        if is_tty:
            tty.setcbreak(fd)
        out.write(E + '[?1049h' + E + '[?25l')
        out.flush()
        if is_tty and not ratty:
            ratty_ok = ratty = detect_ratty(fd)
        registered = False

        yaw, pitch, spin, paused = 0.6, 0.45, True, False
        sample, loads, target = None, [0.0] * n, [0.0] * n
        next_at, last, count = 0.0, time.monotonic(), 0
        W = H = 0
        while True:
            now = time.monotonic()
            dt, last = now - last, now
            if not paused and now >= next_at:
                s = src.next()
                if s is not None:
                    sample, target = s, [float(v) for v in s['cores'][:n]]
                next_at = now + 1.0
            a = 1 - math.exp(-dt / 0.25)
            loads = [l + (t - l) * a for l, t in zip(loads, target)]
            if spin:
                yaw += 0.35 * dt

            if resized[0]:
                resized[0] = False
                W, H = shutil.get_terminal_size()
                out.write(E + '[2J')
            buf = [E + '[H']
            if W < MIN_W or H < MIN_H:
                buf.append(E + '[2J' + E + '[H' + DIM + 'resize to at least %d×%d' % (MIN_W, MIN_H) + R)
            else:
                if ratty and not registered:
                    buf += [rgp('r', id=i + 1, fmt='obj', path=CUBE) for i in range(n)]
                    registered = True
                bw, bh = W - 2, H - 2
                iw, ih = bw - 4, bh - 2
                cam = Camera(yaw, pitch)
                cells, anchors, ticks = render_chart(iw, ih, cam, cores, loads, not ratty)
                for cr, cc, text in ticks:
                    for j, ch in enumerate(text):
                        if 0 <= cr < ih and 0 <= cc + j < iw and cells[cr][cc + j][1] in (None, DIM):
                            cells[cr][cc + j] = (ch, DIM)
                for i, (cr, cc) in enumerate(anchors):  # labels in dim, only over empty or floor cells
                    for j, ch in enumerate(cores[i][2]):
                        if 0 <= cr < ih and 0 <= cc + j < iw and cells[cr][cc + j][1] in (None, DIM):
                            cells[cr][cc + j] = (ch, DIM)
                lines = []
                for row in cells:
                    s, cur = '', None
                    for ch, col in row:
                        if col != cur:
                            s += R + (col or '')
                            cur = col
                        s += ch
                    lines.append(s + R)

                name = '%s%s%s%s  ·  %sE + %sP CPU  ·  %s-core GPU%s' % (BOLD, meta['name'], R, DIM, ne, np_,
                                                                       meta['gpuCores'], R)
                if sample is None:
                    clock, right = MID + '● starting mactop…' + R, DIM + '  —' + R
                else:
                    clock = DIM + ('paused  ' if paused else '') + (sample['t'][11:19] if 'T' in sample['t'] else sample['t']) + R
                    right = BOLD + level(sample['cpu']) + '%.0f%%' % sample['cpu'] + R
                buf.append(E + '[1;1H ' + spread(name, clock, W - 2))
                for r, l in enumerate(box('cores · 3d', right, bw, lines)):
                    buf.append(E + '[%d;2H' % (r + 2) + l)
                rs = 'ratty: on' if ratty else ('ratty: off' if ratty_ok else 'ratty: off (not detected)')
                keys = '←/→ rotate · ↑/↓ tilt · space pause · a auto-spin · r ratty · q quit'
                room = W - 2 - len(rs) - 2
                keys = keys if len(keys) <= room else keys[:room - 1] + '…'
                buf.append(E + '[%d;1H ' % H + DIM + spread(keys, rs, W - 2) + R)

                if ratty:  # one object per core, anchored to its label cell (0-based row/col), sits above it
                    for i, (cr, cc) in enumerate(anchors):
                        row, col = cr + 2, cc + 3  # chart cell -> screen cell (0-based)
                        rows = max(1, round(loads[i] / 100 * max(1, min(cr, ih // 2))))
                        p = dict(id=i + 1, row=row - rows, col=col, w=2, h=rows, depth='%.1f' % (0.2 + loads[i] / 100 * 2.8),
                                 color=HEX[level(loads[i])], animate=0)
                        if placed.get(i + 1) != p:
                            placed[i + 1] = p
                            buf.append(rgp('p', **p))
            out.write(''.join(buf))
            out.flush()

            count += 1
            if frames and count >= frames:
                break
            wait = 1 / FPS - (time.monotonic() - now)
            if is_tty:
                r, _, _ = select.select([fd], [], [], max(0, wait))
                if r:
                    keys = os.read(fd, 64)
                    if keys.startswith(E.encode() + b'_'):
                        continue  # a late RGP reply
                    if b'q' in keys or b'\x03' in keys:
                        break
                    yaw += 0.15 * (keys.count(b'\x1b[C') - keys.count(b'\x1b[D'))
                    pitch = min(1.3, max(0.1, pitch + 0.08 * (keys.count(b'\x1b[A') - keys.count(b'\x1b[B'))))
                    if b' ' in keys:
                        paused = not paused
                    if b'a' in keys:
                        spin = not spin
                    if b'r' in keys:
                        ratty = not ratty
                        if not ratty:
                            out.write(''.join(rgp('d', id=i) for i in placed))
                            placed.clear()
                            registered = False
                        out.write(E + '[2J')
            elif wait > 0:
                time.sleep(wait)
    except KeyboardInterrupt:
        pass
    finally:
        out.write(''.join(rgp('d', id=i) for i in placed) + R + E + '[?25h' + E + '[?1049l')
        out.flush()
        if saved:
            termios.tcsetattr(fd, termios.TCSADRAIN, saved)
        if isinstance(src, Live):
            src.close()


if __name__ == '__main__':
    main()
