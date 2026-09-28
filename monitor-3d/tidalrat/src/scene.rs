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
        let d = D.max(2.5 * floor_half(cores));
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

/// Dot-space scale and offset fitted over every yaw, so the chart doesn't breathe while spinning.
pub fn fit(pitch: f64, cores: &[Core], dw: usize, dh: usize) -> (f64, f64, f64) {
    let (mut x0, mut x1, mut y0, mut y1) = (f64::MAX, f64::MIN, f64::MAX, f64::MIN);
    for k in 0..24 {
        let cam = Camera::new(k as f64 * std::f64::consts::PI / 12.0, pitch, cores);
        for (x, y, z) in extent(&cam, cores) {
            let (sx, sy, _) = cam.view(x, y, z);
            (x0, x1, y0, y1) = (x0.min(sx), x1.max(sx), y0.min(sy), y1.max(sy));
        }
    }
    let k = ((dw as f64 - 4.0) / (x1 - x0)).min((dh as f64 - 6.0) / (y1 - y0));
    (
        k,
        dw as f64 / 2.0 - k * (x1 + x0) / 2.0,
        dh as f64 / 2.0 + k * (y1 + y0) / 2.0,
    )
}

type Pattern = fn(i64, i64) -> bool;
const FULL: Pattern = |_, _| true;
const STRIPE: Pattern = |x, _| x % 2 == 0;
const CHECK: Pattern = |x, y| (x + y) % 2 == 0;

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
    /// scale post ticks: (row, col of the post, text); the caller places the text beside it
    pub ticks: Vec<(i64, i64, &'static str)>,
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
            ((a.1 / 4.0).floor() as i64, (a.0 / 2.0).round() as i64, text)
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
    fn fit_keeps_every_point_inside_and_the_chart_large() {
        // M2 Pro, a small chip, a wide one (M3 Ultra) and a P-only row
        for (ne, np) in [(4, 6), (2, 2), (8, 24), (0, 12)] {
            let cores = core_layout(ne, np);
            let (dw, dh) = (160, 80);
            for pitch in [0.1, 0.45, 1.3] {
                let (k, ox, oy) = fit(pitch, &cores, dw, dh);
                for step in 0..48 {
                    let cam = Camera::new(step as f64 * 0.13, pitch, &cores);
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
}
