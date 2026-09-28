//! Ratty Graphics Protocol: detect Ratty, and hand each core column to it as a real 3D cube
//! anchored to the core's label cell. https://blog.orhun.dev/introducing-ratty/

use crate::theme::{hex, level};
use ratatui::{Frame, layout::Rect};
use ratatui_ratty::{RattyGraphic, RattyGraphicSettings};
use std::io::{self, Read, Write};
use std::time::{Duration, Instant};

const CUBE: &[u8] = include_bytes!("cube.obj");

/// Sends the RGP support query and waits up to 150 ms for `ESC _ ratty;g;s;v=`.
/// Call in raw mode, before anything else reads stdin.
pub fn detect() -> bool {
    let mut out = io::stdout();
    if out
        .write_all(b"\x1b_ratty;g;s\x1b\\")
        .and_then(|_| out.flush())
        .is_err()
    {
        return false;
    }
    let (mut buf, end) = (Vec::new(), Instant::now() + Duration::from_millis(150));
    while let Some(left) = end.checked_duration_since(Instant::now()) {
        let mut fd = libc::pollfd {
            fd: 0,
            events: libc::POLLIN,
            revents: 0,
        };
        if unsafe { libc::poll(&mut fd, 1, left.as_millis() as i32) } <= 0 {
            break;
        }
        let mut chunk = [0u8; 256];
        match io::stdin().read(&mut chunk) {
            Ok(n) if n > 0 => buf.extend_from_slice(&chunk[..n]),
            _ => break,
        }
        if buf.windows(2).any(|w| w == b"\x1b\\") {
            break;
        }
    }
    buf.windows(15).any(|w| w == b"\x1b_ratty;g;s;v=")
}

pub struct Columns {
    graphics: Vec<RattyGraphic<'static>>,
    pub on: bool,
}

impl Columns {
    pub fn new(n: usize) -> Self {
        let graphics = (0..n)
            .map(|i| {
                RattyGraphic::new(
                    RattyGraphicSettings::new("cube.obj")
                        .id(i as u32 + 1)
                        .animate(false),
                )
            })
            .collect();
        Columns {
            graphics,
            on: false,
        }
    }

    pub fn set(&mut self, on: bool) -> io::Result<()> {
        if on != self.on {
            for g in &self.graphics {
                if on {
                    g.register_payload_with_name(CUBE, Some("cube.obj"))?
                } else {
                    g.clear()?
                }
            }
            self.on = on;
        }
        Ok(())
    }

    /// Place one column per core on its label cell, `rows` tall at 100%.
    pub fn render(&mut self, frame: &mut Frame, anchors: &[(u16, u16)], loads: &[f64], rows: u16) {
        if !self.on {
            return;
        }
        let area = frame.area();
        for (g, (&(row, col), &load)) in self.graphics.iter_mut().zip(anchors.iter().zip(loads)) {
            if row == 0 {
                continue;
            }
            let h = ((load / 100.0 * rows as f64).round() as u16).clamp(1, row);
            let rect = Rect::new(col, row - h, 2, h).intersection(area);
            let s = g.settings_mut();
            s.color = Some(hex(level(load)));
            // rounded so the place message is only re-sent when the column visibly changes
            s.depth = ((0.2 + load / 100.0 * 2.8) * 10.0).round() as f32 / 10.0;
            frame.render_widget(&*g, rect);
        }
    }
}
