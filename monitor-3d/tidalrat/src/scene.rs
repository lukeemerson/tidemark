//! Cores as a 3D bar chart rasterised into braille (2×4 dots per cell), ported from
//! prototypes/e-terminal-3d/monitor3d.py. Painter's order: far columns first, nearer ones
//! overwrite their dots, and each cell takes the colour of its frontmost column.

use crate::theme::{Role, level};

pub const HMAX: f64 = 3.0; // column height at 100% (world units)
const D: f64 = 10.0; // minimum camera distance
const S: f64 = 0.3; // column half-width
const BITS: [[u32; 2]; 4] = [[0x01, 0x08], [0x02, 0x10], [0x04, 0x20], [0x40, 0x80]]; // [dy][dx]

#[derive(Clone, Copy)]
pub struct Camera {
    pub yaw: f64,
    pub pitch: f64,
    d: f64,
    cy: f64,
    sy: f64,
    cp: f64,
    sp: f64,
    pos: [f64; 3],
}

impl Camera {
    /// The distance grows with the layout so wide chips (8E + 24P) stay in front of the camera.
    pub fn new(yaw: f64, pitch: f64, cores: &[Core]) -> Self {
        Camera::at(yaw, pitch, D.max(2.5 * floor_half(cores)))
    }

    /// A camera `d` from the rotation axis.
    fn at(yaw: f64, pitch: f64, d: f64) -> Self {
        let (cy, sy, cp, sp) = (yaw.cos(), yaw.sin(), pitch.cos(), pitch.sin());
        let pos = [d * cp * sy, HMAX * 0.4 + d * sp, d * cp * cy];
        Camera {
            yaw,
            pitch,
            d,
            cy,
            sy,
            cp,
            sp,
            pos,
        }
    }

    /// World point -> (screen x, screen y up, distance).
    fn view(&self, x: f64, y: f64, z: f64) -> (f64, f64, f64) {
        let y = y - HMAX * 0.4;
        let x1 = x * self.cy - z * self.sy;
        let z1 = x * self.sy + z * self.cy;
        let y2 = y * self.cp - z1 * self.sp;
        let z2 = y * self.sp + z1 * self.cp;
        let d = self.d - z2;
        (x1 / d, y2 / d, d)
    }
}

pub struct Core {
    pub x: f64,
    pub z: f64,
    pub label: String,
}

/// E cores in the back row, P cores in front.
pub fn core_layout(ne: usize, np: usize) -> Vec<Core> {
    let row = |n: usize, z: f64, tag: &str| {
        (0..n)
            .map(move |i| Core {
                x: i as f64 - (n as f64 - 1.0) / 2.0,
                z,
                label: format!("{tag}{}", i + 1),
            })
            .collect::<Vec<_>>()
    };
    let mut v = row(ne, -0.6, "E");
    v.extend(row(np, 0.6, "P"));
    v
}

fn corners(x: f64, z: f64) -> [(f64, f64); 4] {
    [
        (x - S, z - S),
        (x + S, z - S),
        (x + S, z + S),
        (x - S, z + S),
    ]
}

fn label_point(cam: &Camera, x: f64, z: f64) -> (f64, f64, f64) {
    (x + 0.62 * cam.yaw.sin(), 0.0, z + 0.62 * cam.yaw.cos())
}

/// Half-width of the floor grid: half a slot past the outermost column, so grid lines fall
/// between columns.
fn floor_half(cores: &[Core]) -> f64 {
    cores.iter().map(|c| c.x.abs()).fold(0.0, f64::max) + 0.5
}

/// Every world point the chart can draw at this yaw: column corners, the label point, floor
/// corners and the scale post's top.
fn extent(cam: &Camera, cores: &[Core]) -> Vec<(f64, f64, f64)> {
    let h = floor_half(cores);
    let mut pts = vec![
        (-h, 0.0, -1.2),
        (h, 0.0, -1.2),
        (-h, 0.0, 1.2),
        (h, 0.0, 1.2),
        (-h, HMAX, -1.2),
    ];
    for c in cores {
        pts.extend(
            corners(c.x, c.z)
                .iter()
                .flat_map(|&(a, b)| [(a, 0.0, b), (a, HMAX, b)]),
        );
        pts.push(label_point(cam, c.x, c.z));
    }
    pts
}

/// Dot-space scale and offset for a `dw`×`dh` canvas, the same at every yaw so the chart spins in
/// place: no change of size while turning, and the rotation axis (world x = z = 0, which projects
/// to screen x = 0 at any yaw) pinned to the centre column. The scale is the largest that keeps
/// every drawn point inside at every yaw (sampled every 5°), measured out from that axis.
pub fn fit(pitch: f64, cores: &[Core], dw: usize, dh: usize) -> (f64, f64, f64) {
    let d = Camera::new(0.0, pitch, cores).d;
    fit_extent(pitch, d, dw, dh, |cam| extent(cam, cores))
}

/// `fit` for any scene: `extent` lists every point the scene can draw from a given camera.
fn fit_extent(
    pitch: f64,
    d: f64,
    dw: usize,
    dh: usize,
    extent: impl Fn(&Camera) -> Vec<(f64, f64, f64)>,
) -> (f64, f64, f64) {
    let (mut ax, mut y0, mut y1) = (0.0f64, f64::MAX, f64::MIN);
    for step in 0..72 {
        let cam = Camera::at(step as f64 * std::f64::consts::PI / 36.0, pitch, d);
        for (x, y, z) in extent(&cam) {
            let (sx, sy, _) = cam.view(x, y, z);
            (ax, y0, y1) = (ax.max(sx.abs()), y0.min(sy), y1.max(sy));
        }
    }
    let k = ((dw as f64 / 2.0 - 2.0) / ax).min((dh as f64 - 6.0) / (y1 - y0));
    (k, dw as f64 / 2.0, dh as f64 / 2.0 + k * (y1 + y0) / 2.0)
}

type Pattern = fn(i64, i64) -> bool;
const FULL: Pattern = |_, _| true;
const STRIPE: Pattern = |x, _| x % 2 == 0;
const CHECK: Pattern = |x, y| (x + y) % 4 == 0; // 2 of 8 dots: shaded reads darker than STRIPE's 4

struct Canvas {
    cw: usize,
    ch: usize,
    dw: usize,
    dh: usize,
    on: Vec<bool>,
    owner: Vec<i32>, // draw rank of the column that owns the dot, -1 = none
    floor: Vec<bool>,
}

impl Canvas {
    fn new(cw: usize, ch: usize) -> Self {
        let n = cw * 2 * ch * 4;
        Canvas {
            cw,
            ch,
            dw: cw * 2,
            dh: ch * 4,
            on: vec![false; n],
            owner: vec![-1; n],
            floor: vec![false; n],
        }
    }

    fn put(&mut self, x: i64, y: i64, v: bool, owner: i32) {
        if x < 0 || y < 0 || x >= self.dw as i64 || y >= self.dh as i64 {
            return;
        }
        let i = y as usize * self.dw + x as usize;
        if owner < 0 {
            self.floor[i] = true;
        } else {
            self.on[i] = v;
            self.owner[i] = owner;
        }
    }

    fn line(&mut self, a: (f64, f64), b: (f64, f64), owner: i32) {
        let (mut x0, mut y0, x1, y1) = (
            a.0.round() as i64,
            a.1.round() as i64,
            b.0.round() as i64,
            b.1.round() as i64,
        );
        let (dx, dy) = ((x1 - x0).abs(), -(y1 - y0).abs());
        let (sx, sy) = (if x0 < x1 { 1 } else { -1 }, if y0 < y1 { 1 } else { -1 });
        let mut err = dx + dy;
        loop {
            self.put(x0, y0, true, owner);
            if x0 == x1 && y0 == y1 {
                return;
            }
            let e2 = 2 * err;
            if e2 >= dy {
                err += dy;
                x0 += sx;
            }
            if e2 <= dx {
                err += dx;
                y0 += sy;
            }
        }
    }

    fn poly(&mut self, pts: &[(f64, f64)], pat: Pattern, owner: i32) {
        let (ymin, ymax) = pts
            .iter()
            .fold((f64::MAX, f64::MIN), |(a, b), p| (a.min(p.1), b.max(p.1)));
        let (ya, yb) = (
            ((ymin - 0.5).ceil() as i64).max(0),
            ((ymax - 0.5).floor() as i64).min(self.dh as i64 - 1),
        );
        for y in ya..=yb {
            let yc = y as f64 + 0.5;
            let mut xs = vec![];
            for (i, &(ax, ay)) in pts.iter().enumerate() {
                let (bx, by) = pts[(i + 1) % pts.len()];
                if (ay <= yc && yc < by) || (by <= yc && yc < ay) {
                    xs.push(ax + (yc - ay) * (bx - ax) / (by - ay));
                }
            }
            if xs.len() < 2 {
                continue;
            }
            let (xmin, xmax) = xs
                .iter()
                .fold((f64::MAX, f64::MIN), |(a, b), &x| (a.min(x), b.max(x)));
            for x in ((xmin - 0.5).ceil() as i64).max(0)
                ..=((xmax - 0.5).floor() as i64).min(self.dw as i64 - 1)
            {
                self.put(x, y, pat(x, y), owner);
            }
        }
    }

    fn cells(&self, colours: &[Role]) -> Vec<Vec<(char, Option<Role>)>> {
        (0..self.ch)
            .map(|cy| {
                (0..self.cw)
                    .map(|cx| {
                        let (mut bits, mut fbits, mut best) = (0u32, 0u32, -1);
                        for (dy, row) in BITS.iter().enumerate() {
                            for (dx, bit) in row.iter().enumerate() {
                                let i = (cy * 4 + dy) * self.dw + cx * 2 + dx;
                                if self.owner[i] >= 0 {
                                    if self.on[i] {
                                        bits |= bit;
                                    }
                                    best = best.max(self.owner[i]);
                                } else if self.floor[i] {
                                    fbits |= bit;
                                }
                            }
                        }
                        let braille = |b| char::from_u32(0x2800 + b).unwrap();
                        if best >= 0 {
                            (
                                if bits > 0 { braille(bits) } else { ' ' },
                                Some(colours[best as usize]),
                            )
                        } else if fbits > 0 {
                            (braille(fbits), Some(Role::Dim))
                        } else {
                            (' ', None)
                        }
                    })
                    .collect()
            })
            .collect()
    }
}

pub struct Chart {
    pub cells: Vec<Vec<(char, Option<Role>)>>,
    /// per core: (row, col) cell where its label starts, in chart cells
    pub labels: Vec<(i64, i64)>,
    /// scale ticks: (row, col of the anchor, text); the caller places the text beside it
    pub ticks: Vec<(i64, i64, String)>,
    /// single glyphs drawn on top of everything: (row, col, glyph, colour)
    pub marks: Vec<(i64, i64, char, Role)>,
}

/// `columns = false` draws only the floor, labels and scale (Ratty draws the columns in 3D).
pub fn render(
    cw: usize,
    ch: usize,
    cam: &Camera,
    cores: &[Core],
    loads: &[f64],
    columns: bool,
) -> Chart {
    let mut cv = Canvas::new(cw, ch);
    let (k, ox, oy) = fit(cam.pitch, cores, cv.dw, cv.dh);
    let proj = |x: f64, y: f64, z: f64| {
        let (sx, sy, _) = cam.view(x, y, z);
        (ox + sx * k, oy - sy * k)
    };
    let half = floor_half(cores);

    // floor grid in dim
    let mut gx = -half;
    while gx <= half + 1e-9 {
        cv.line(proj(gx, 0.0, -1.2), proj(gx, 0.0, 1.2), -1);
        gx += 1.0;
    }
    for gz in [-1.2, 0.0, 1.2] {
        cv.line(proj(-half, 0.0, gz), proj(half, 0.0, gz), -1);
    }

    // scale post at the back-left corner: 50% and 100% of HMAX, so headroom reads as scale
    let ticks = [(0.5, "50%"), (1.0, "100%")]
        .iter()
        .map(|&(f, text)| {
            let a = proj(-half, HMAX * f, -1.2);
            (
                (a.1 / 4.0).floor() as i64,
                (a.0 / 2.0).round() as i64,
                text.to_string(),
            )
        })
        .collect();
    let post = |cv: &mut Canvas, owner: i32| {
        cv.line(proj(-half, 0.0, -1.2), proj(-half, HMAX, -1.2), owner);
        for f in [0.5, 1.0] {
            let a = proj(-half, HMAX * f, -1.2);
            cv.line(a, proj(-half + 0.3, HMAX * f, -1.2), owner);
        }
    };

    let labels = cores
        .iter()
        .map(|c| {
            let (lx, ly, lz) = label_point(cam, c.x, c.z);
            let (px, py) = proj(lx, ly, lz);
            ((py / 4.0).floor() as i64, (px / 2.0).round() as i64 - 1)
        })
        .collect();

    let mut colours = vec![];
    if !columns {
        post(&mut cv, -1);
    } else {
        // columns and the post in one far-to-near order, so nearer columns hide the post and
        // the post hides columns behind it; `None` is the post
        let dist = |x: f64, z: f64| cam.view(x, 0.0, z).2;
        let mut order: Vec<Option<usize>> = (0..cores.len()).map(Some).chain([None]).collect();
        order.sort_by(|a, b| {
            let d =
                |o: &Option<usize>| o.map_or(dist(-half, -1.2), |i| dist(cores[i].x, cores[i].z));
            d(b).total_cmp(&d(a))
        });
        let (lx, lz) = (-0.55, 0.83); // side light, world x/z
        for (rank, &o) in order.iter().enumerate() {
            let Some(i) = o else {
                colours.push(Role::Dim);
                post(&mut cv, rank as i32);
                continue;
            };
            let c = &cores[i];
            let load = loads.get(i).copied().unwrap_or(0.0);
            let h = (load / 100.0 * HMAX).max(0.04);
            let cn = corners(c.x, c.z);
            let bot: Vec<_> = cn.iter().map(|&(a, b)| proj(a, 0.0, b)).collect();
            let top: Vec<_> = cn.iter().map(|&(a, b)| proj(a, h, b)).collect();
            let mut faces = vec![([0.0, 1.0, 0.0], top.clone(), [c.x, h, c.z])];
            for ((a, b), n) in [
                ((3, 2), [0.0, 0.0, 1.0]),
                ((0, 1), [0.0, 0.0, -1.0]),
                ((1, 2), [1.0, 0.0, 0.0]),
                ((0, 3), [-1.0, 0.0, 0.0]),
            ] {
                faces.push((
                    n,
                    vec![bot[a], bot[b], top[b], top[a]],
                    [c.x + n[0] * S, h / 2.0, c.z + n[2] * S],
                ));
            }
            colours.push(level(load));
            for (n, pts, fc) in faces {
                let facing: f64 = (0..3).map(|j| n[j] * (cam.pos[j] - fc[j])).sum();
                if facing <= 0.0 {
                    continue; // back face
                }
                let pat = if n[1] > 0.0 {
                    FULL
                } else if n[0] * lx + n[2] * lz > 0.0 {
                    STRIPE
                } else {
                    CHECK
                };
                cv.poly(&pts, pat, rank as i32);
                for j in 0..pts.len() {
                    cv.line(pts[j], pts[(j + 1) % pts.len()], rank as i32);
                }
            }
        }
    }
    Chart {
        cells: cv.cells(&colours),
        labels,
        ticks,
        marks: vec![],
    }
}

/// Cores × time as a braille landscape: one ridge per core, newest sample at the front edge.
/// Far ridges are drawn first, and the area under each ridge is erased down to the floor, so
/// nearer ridges hide what's behind them. Each 1-sample segment takes its newer end's level
/// colour. Framing is fixed across yaws with the rotation axis centred, like `fit`.
pub struct Terrain<'a> {
    /// cores in lane order (E first)
    pub lanes: usize,
    /// samples in the window, oldest first, one load per core; at most `window` long, and
    /// aligned so the last row sits at the front edge
    pub rows: &'a [Vec<f64>],
    pub window: usize,
    /// rows (indices into `rows`) where an alert rule started firing, and whether that row is
    /// the cursor's (the front row)
    pub alerts: &'a [(usize, bool)],
}

const TERRAIN_MIN_DEPTH: f64 = 6.0;
/// alert markers sit this many lane widths outside the floor's side edge, clear of the lanes
const ALERT_GAP: f64 = 2.5;

impl Terrain<'_> {
    fn half_x(&self) -> f64 {
        self.lanes as f64 / 2.0
    }
    fn half_z(&self) -> f64 {
        self.lanes.max(TERRAIN_MIN_DEPTH as usize) as f64 / 2.0
    }
    fn lane_x(&self, c: usize) -> f64 {
        c as f64 - (self.lanes as f64 - 1.0) / 2.0
    }
    /// z of window slot `j` (0 = oldest slot, window-1 = front edge)
    fn slot_z(&self, j: usize) -> f64 {
        let hz = self.half_z();
        -hz + 2.0 * hz * j as f64 / (self.window.max(2) - 1) as f64
    }
    fn distance(&self) -> f64 {
        D.max(2.5 * self.half_x().max(self.half_z()))
    }
    fn label_point(&self, c: usize) -> (f64, f64, f64) {
        (self.lane_x(c), 0.0, self.half_z() + 0.6)
    }
    fn extent(&self) -> Vec<(f64, f64, f64)> {
        let (hx, hz) = (self.half_x(), self.half_z());
        // the alert marker lines beside both side edges, so markers never leave the frame
        let ax = hx + ALERT_GAP;
        let mut pts = vec![
            (-ax, 0.0, -hz),
            (-ax, 0.0, hz),
            (ax, 0.0, -hz),
            (ax, 0.0, hz),
        ];
        for x in [-hx, hx] {
            for z in [-hz, hz + 0.6] {
                for y in [0.0, HMAX] {
                    pts.push((x, y, z));
                }
            }
        }
        pts
    }

    pub fn camera(&self, yaw: f64, pitch: f64) -> Camera {
        Camera::at(yaw, pitch, self.distance())
    }

    pub fn render(&self, cw: usize, ch: usize, cam: &Camera) -> Chart {
        let mut cv = Canvas::new(cw, ch);
        let pts = self.extent();
        let (k, ox, oy) = fit_extent(cam.pitch, self.distance(), cv.dw, cv.dh, |_| pts.clone());
        let proj = |x: f64, y: f64, z: f64| {
            let (sx, sy, _) = cam.view(x, y, z);
            (ox + sx * k, oy - sy * k)
        };
        let (hx, hz) = (self.half_x(), self.half_z());

        // floor in dim: the outline, and a line every 10 samples back from the front edge
        for (a, b) in [
            ((-hx, -hz), (hx, -hz)),
            ((-hx, hz), (hx, hz)),
            ((-hx, -hz), (-hx, hz)),
            ((hx, -hz), (hx, hz)),
        ] {
            cv.line(proj(a.0, 0.0, a.1), proj(b.0, 0.0, b.1), -1);
        }
        let mut j = self.window as i64 - 1 - 10;
        while j > 0 {
            let z = self.slot_z(j as usize);
            cv.line(proj(-hx, 0.0, z), proj(hx, 0.0, z), -1);
            j -= 10;
        }

        let off = self.window.saturating_sub(self.rows.len());
        let mut order: Vec<usize> = (0..self.lanes).collect();
        let dist = |c: usize| cam.view(self.lane_x(c), 0.0, 0.0).2;
        order.sort_by(|&a, &b| dist(b).total_cmp(&dist(a)));
        let mut colours = vec![];
        let never: Pattern = |_, _| false;
        for &c in &order {
            let x = self.lane_x(c);
            let h = |r: &Vec<f64>| r.get(c).copied().unwrap_or(0.0) / 100.0 * HMAX;
            for (i, pair) in self.rows.windows(2).enumerate() {
                let (z0, z1) = (self.slot_z(off + i), self.slot_z(off + i + 1));
                let (h0, h1) = (h(&pair[0]), h(&pair[1]));
                let rank = colours.len() as i32;
                colours.push(level(pair[1].get(c).copied().unwrap_or(0.0)));
                let (p0, p1) = (proj(x, h0, z0), proj(x, h1, z1));
                cv.poly(&[proj(x, 0.0, z0), proj(x, 0.0, z1), p1, p0], never, rank);
                cv.line(p0, p1, rank);
            }
            // the front edge drops to the floor, so each lane reads as a solid profile
            if let Some(last) = self.rows.last() {
                let z = self.slot_z(self.window - 1);
                let rank = colours.len() as i32;
                colours.push(level(last.get(c).copied().unwrap_or(0.0)));
                cv.line(proj(x, 0.0, z), proj(x, h(last), z), rank);
            }
        }

        let labels = (0..self.lanes)
            .map(|c| {
                let (lx, ly, lz) = self.label_point(c);
                let (px, py) = proj(lx, ly, lz);
                ((py / 4.0).floor() as i64, (px / 2.0).round() as i64 - 1)
            })
            .collect();
        // ▲ on the floor plane beside whichever side edge faces screen-left at this yaw, level
        // with the row where a rule fired; it switches sides as the terrain turns past side-on
        let ax = hx + ALERT_GAP;
        let mx = if cam.view(-ax, 0.0, 0.0).0 <= cam.view(ax, 0.0, 0.0).0 {
            -ax
        } else {
            ax
        };
        let marks = self
            .alerts
            .iter()
            .filter(|&&(r, _)| r < self.rows.len())
            .map(|&(r, on_cursor)| {
                let (px, py) = proj(mx, 0.0, self.slot_z(off + r));
                let role = if on_cursor { Role::Mid } else { Role::High };
                (
                    (py / 4.0).floor() as i64,
                    (px / 2.0).floor() as i64,
                    '▲',
                    role,
                )
            })
            .collect();
        Chart {
            cells: cv.cells(&colours),
            labels,
            ticks: vec![],
            marks,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn layout_puts_e_behind_p() {
        let c = core_layout(4, 6);
        assert_eq!(c.len(), 10);
        assert_eq!((c[0].label.as_str(), c[9].label.as_str()), ("E1", "P6"));
        assert!(c[0].z < c[4].z);
    }

    #[test]
    fn fit_spins_in_place_keeps_every_point_inside_and_the_chart_large() {
        // M2 Pro, a small chip, a wide one (M3 Ultra) and a P-only row
        for (ne, np) in [(4, 6), (2, 2), (8, 24), (0, 12)] {
            let cores = core_layout(ne, np);
            let (dw, dh) = (160, 80);
            for pitch in [0.1, 0.45, 1.3] {
                let (k, ox, oy) = fit(pitch, &cores, dw, dh);
                // spins in place: the rotation axis sits on the centre column at every yaw
                assert_eq!(ox, dw as f64 / 2.0);
                // yaws off the 5° sampling grid, to catch overshoot between samples
                for step in 0..48 {
                    let cam = Camera::new(step as f64 * 0.13, pitch, &cores);
                    let (ax, _, _) = cam.view(0.0, HMAX, 0.0);
                    assert!((ox + ax * k - dw as f64 / 2.0).abs() < 1e-9);
                    for (x, y, z) in extent(&cam, &cores) {
                        let (sx, sy, d) = cam.view(x, y, z);
                        assert!(d > 1.0, "{ne}E+{np}P: point behind the camera (d={d})");
                        let (px, py) = (ox + sx * k, oy - sy * k);
                        assert!(
                            px > 0.0 && px < dw as f64 && py > 0.0 && py < dh as f64,
                            "{ne}E+{np}P pitch {pitch}: ({px},{py}) outside {dw}x{dh}"
                        );
                    }
                }
                // the column footprint seen head-on spans a real share of the canvas's short side,
                // not a dot (the wide-chip collapse left it at about 1 dot)
                let cam = Camera::new(0.0, pitch, &cores);
                let xs: Vec<f64> = cores
                    .iter()
                    .map(|c| ox + cam.view(c.x, 0.0, c.z).0 * k)
                    .collect();
                let span = xs.iter().cloned().fold(f64::MIN, f64::max)
                    - xs.iter().cloned().fold(f64::MAX, f64::min);
                assert!(
                    span > dw.min(dh) as f64 * 0.1,
                    "{ne}E+{np}P pitch {pitch}: columns span only {span} dots"
                );
            }
        }
    }

    #[test]
    fn grid_lines_fall_between_columns() {
        let cores = core_layout(4, 6);
        let h = floor_half(&cores);
        assert_eq!(h, 3.0); // lines at -3..3, columns at -2.5..2.5
    }

    #[test]
    fn braille_bits_and_frontmost_colour() {
        let mut cv = Canvas::new(1, 1);
        cv.put(0, 3, true, 0); // bottom-left dot
        cv.put(1, 3, true, 1); // bottom-right dot, nearer column
        let cells = cv.cells(&[Role::Low, Role::High]);
        assert_eq!(cells[0][0], ('\u{28C0}', Some(Role::High)));
    }

    #[test]
    fn full_load_columns_are_drawn_in_their_level() {
        let cores = core_layout(4, 6);
        let chart = render(
            80,
            24,
            &Camera::new(0.6, 0.45, &cores),
            &cores,
            &[90.0; 10],
            true,
        );
        let roles: Vec<_> = chart.cells.iter().flatten().filter_map(|c| c.1).collect();
        assert!(roles.contains(&Role::High));
        assert!(!roles.contains(&Role::Low));
    }

    #[test]
    fn terrain_fits_every_yaw_and_colours_by_level() {
        let rows: Vec<Vec<f64>> = (0..90)
            .map(|i| vec![if i > 60 { 95.0 } else { 20.0 }; 10])
            .collect();
        let t = Terrain {
            lanes: 10,
            rows: &rows,
            window: 90,
            alerts: &[],
        };
        let (dw, dh) = (160, 96);
        let (k, ox, oy) = fit_extent(0.45, t.distance(), dw, dh, |_| t.extent());
        assert_eq!(ox, dw as f64 / 2.0);
        for step in 0..48 {
            let cam = t.camera(step as f64 * 0.13, 0.45);
            for (x, y, z) in t.extent() {
                let (sx, sy, d) = cam.view(x, y, z);
                assert!(d > 1.0);
                let (px, py) = (ox + sx * k, oy - sy * k);
                assert!(
                    px > 0.0 && px < dw as f64 && py > 0.0 && py < dh as f64,
                    "({px},{py})"
                );
            }
        }
        let chart = t.render(80, 24, &t.camera(0.6, 0.45));
        let roles: Vec<_> = chart.cells.iter().flatten().filter_map(|c| c.1).collect();
        assert!(
            roles.contains(&Role::High) && roles.contains(&Role::Low) && roles.contains(&Role::Dim)
        );
        assert_eq!(chart.labels.len(), 10);
    }

    #[test]
    fn terrain_with_a_short_history_sits_at_the_front() {
        let rows = vec![vec![50.0; 10]; 5];
        let t = Terrain {
            lanes: 10,
            rows: &rows,
            window: 90,
            alerts: &[],
        };
        assert!(t.slot_z(89) > t.slot_z(0));
        let chart = t.render(80, 24, &t.camera(0.6, 0.45));
        assert!(chart.cells.iter().flatten().any(|c| c.1 == Some(Role::Mid)));
    }

    #[test]
    fn alert_markers_sit_beside_the_floor_and_follow_the_cursor() {
        let rows = vec![vec![30.0; 10]; 90];
        let alerts = [(10, false), (89, true)];
        let t = Terrain {
            lanes: 10,
            rows: &rows,
            window: 90,
            alerts: &alerts,
        };
        let chart = t.render(100, 30, &t.camera(0.0, 0.45));
        assert_eq!(chart.marks.len(), 2);
        // markers take whichever side edge faces screen-left, switching sides past side-on
        for yaw in [0.6, -0.6, 2.2, 3.0, 4.0] {
            let c = t.render(100, 30, &t.camera(yaw, 0.45));
            assert!(c.marks.iter().all(|m| m.1 < 50), "yaw {yaw}: {:?}", c.marks);
        }
        assert_eq!(chart.marks[0].3, Role::High);
        assert_eq!(chart.marks[1].3, Role::Mid); // the cursor's row
        // at the default view each marker sits left of everything the terrain draws on its row
        for &(r, c, _, _) in &chart.marks {
            let row = &chart.cells[r as usize];
            if let Some(left) = row.iter().position(|cell| cell.1.is_some()) {
                assert!(
                    c < left as i64,
                    "marker at col {c} overlaps the terrain (row {r} starts at {left})"
                );
            }
        }
        // an alert row past `rows` is ignored
        let t2 = Terrain {
            alerts: &[(200, false)],
            ..t
        };
        assert!(t2.render(100, 30, &t2.camera(0.0, 0.45)).marks.is_empty());
    }
}
