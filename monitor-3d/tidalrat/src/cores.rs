//! `tidalrat --cores`: per-core load as a rotating 3D bar chart inside a Monitor TUI heavy frame.

use crate::ratty::{self, Columns};
use crate::scene::{self, Camera};
use crate::source::{Meta, Sample, Source};
use crate::theme::{Role, bold, fg, level};
use ratatui::{
    DefaultTerminal, Frame,
    buffer::Buffer,
    crossterm::event::{self, Event, KeyCode, KeyEventKind, KeyModifiers},
    layout::Rect,
    text::{Line, Span},
    widgets::{Block, BorderType, Paragraph, Widget},
};
use std::io;
use std::time::{Duration, Instant};

pub struct Opts {
    pub replay: bool,
    pub ratty: bool,
    pub frames: Option<u64>,
}

const FRAME: Duration = Duration::from_millis(50);
const MIN_W: u16 = 60;
const MIN_H: u16 = 20;

struct App {
    source: Source,
    meta: Option<Meta>,
    sample: Option<Sample>,
    loads: Vec<f64>,
    yaw: f64,
    pitch: f64,
    spin: bool,
    paused: bool,
    ratty_found: bool,
    columns: Columns,
}

pub fn run(opts: Opts) -> io::Result<()> {
    let source = if opts.replay {
        Source::replay()
    } else {
        Source::live().unwrap_or_else(Source::replay)
    };
    let meta = if source.is_replay() {
        Some(crate::source::recording().0)
    } else {
        crate::source::static_meta()
    };
    let mut terminal = ratatui::init();
    let ratty_found = ratty::detect();
    // wipe any echo of the probe in terminals that don't swallow APC (terminal.clear() would
    // query the cursor position, which not every terminal answers)
    io::Write::write_all(&mut io::stdout(), b"\x1b[2J")?;
    let n = meta.as_ref().map_or(10, |m| m.e + m.p);
    let mut app = App {
        source,
        meta,
        sample: None,
        loads: vec![0.0; n],
        yaw: 0.6,
        pitch: 0.45,
        spin: true,
        paused: false,
        ratty_found,
        columns: Columns::new(n),
    };
    let result = app
        .columns
        .set(opts.ratty || ratty_found)
        .and_then(|_| app.run(&mut terminal, opts.frames));
    let _ = app.columns.set(false);
    ratatui::restore();
    result
}

impl App {
    fn run(&mut self, terminal: &mut DefaultTerminal, frames: Option<u64>) -> io::Result<()> {
        let (mut last, mut drawn) = (Instant::now(), 0u64);
        loop {
            let dt = last.elapsed().as_secs_f64();
            last = Instant::now();
            if let Some((s, m)) = self.source.poll() {
                if let Some(m) = m
                    && self.meta.as_ref() != Some(&m)
                {
                    self.loads.resize(m.e + m.p, 0.0);
                    let on = self.columns.on;
                    self.columns.set(false)?;
                    self.columns = Columns::new(m.e + m.p);
                    self.columns.set(on)?;
                    self.meta = Some(m);
                }
                if !self.paused {
                    self.sample = Some(s);
                }
            }
            // heights ease toward the newest sample (~0.25 s time constant)
            let a = 1.0 - (-dt / 0.25).exp();
            if let Some(s) = &self.sample {
                for (l, t) in self.loads.iter_mut().zip(&s.cores) {
                    *l += (t - *l) * a;
                }
            }
            if self.spin && !self.paused {
                self.yaw += 0.35 * dt;
            }
            terminal.draw(|f| self.draw(f))?;
            drawn += 1;
            if frames.is_some_and(|n| drawn >= n) {
                return Ok(());
            }
            let deadline = last + FRAME;
            while let Some(left) = deadline.checked_duration_since(Instant::now()) {
                if !event::poll(left)? {
                    break;
                }
                if let Event::Key(k) = event::read()? {
                    if k.kind != KeyEventKind::Press {
                        continue;
                    }
                    match k.code {
                        KeyCode::Char('q') | KeyCode::Esc => return Ok(()),
                        KeyCode::Char('c') if k.modifiers.contains(KeyModifiers::CONTROL) => {
                            return Ok(());
                        }
                        KeyCode::Left => self.yaw -= 0.15,
                        KeyCode::Right => self.yaw += 0.15,
                        KeyCode::Up => self.pitch = (self.pitch + 0.08).min(1.3),
                        KeyCode::Down => self.pitch = (self.pitch - 0.08).max(0.1),
                        KeyCode::Char(' ') => self.paused = !self.paused,
                        KeyCode::Char('a') => self.spin = !self.spin,
                        KeyCode::Char('r') => self.columns.set(!self.columns.on)?,
                        _ => {}
                    }
                }
            }
        }
    }

    fn draw(&mut self, f: &mut Frame) {
        let area = f.area();
        if area.width < MIN_W || area.height < MIN_H {
            let msg = format!("resize to at least {MIN_W}×{MIN_H}");
            Paragraph::new(Span::styled(msg, fg(Role::Dim)))
                .centered()
                .render(
                    Rect {
                        y: area.height / 2,
                        height: 1,
                        ..area
                    },
                    f.buffer_mut(),
                );
            return;
        }
        let (w, h) = (area.width, area.height);
        self.header(f.buffer_mut(), Rect::new(0, 0, w, 1));

        let right = match &self.sample {
            Some(s) => Span::styled(format!(" {:.0}% ", s.cpu), bold(level(s.cpu))),
            None => Span::styled(" — ", fg(Role::Dim)),
        };
        let block = Block::bordered()
            .border_type(BorderType::Thick)
            .border_style(fg(Role::Dim))
            .title(Line::from(vec![
                Span::styled("━", fg(Role::Dim)),
                Span::styled(" cores · 3d ", bold(Role::Text)),
            ]))
            .title(Line::from(vec![right, Span::styled("━", fg(Role::Dim))]).right_aligned());
        let frame_area = Rect::new(0, 1, w, h - 2);
        let inner = block.inner(frame_area);
        block.render(frame_area, f.buffer_mut());
        let chart_area = Rect {
            x: inner.x + 1,
            width: inner.width.saturating_sub(2),
            ..inner
        };

        let cores = match &self.meta {
            Some(m) => scene::core_layout(m.e, m.p),
            None => scene::core_layout(0, self.loads.len()),
        };
        let cam = Camera::new(self.yaw, self.pitch);
        let chart = scene::render(
            chart_area.width as usize,
            chart_area.height as usize,
            &cam,
            &cores,
            &self.loads,
            !self.columns.on,
        );
        draw_chart(f.buffer_mut(), chart_area, &chart, &cores);

        let anchors: Vec<(u16, u16)> = chart
            .labels
            .iter()
            .map(|&(r, c)| {
                (
                    (chart_area.y as i64 + r).max(0) as u16,
                    (chart_area.x as i64 + c).max(0) as u16,
                )
            })
            .collect();
        let loads = self.loads.clone();
        self.columns
            .render(f, &anchors, &loads, chart_area.height / 2);

        self.legend(f.buffer_mut(), Rect::new(0, h - 1, w, 1));
    }

    fn header(&self, buf: &mut Buffer, area: Rect) {
        let mut left = vec![];
        if let Some(m) = &self.meta {
            let gpu = m.gpu_cores.map_or("?".into(), |g| g.to_string());
            left.push(Span::styled(m.name.clone(), bold(Role::Text)));
            left.push(Span::styled(
                format!("  ·  {}E + {}P CPU  ·  {gpu}-core GPU", m.e, m.p),
                fg(Role::Dim),
            ));
        }
        let right = match &self.sample {
            None => Line::from(Span::styled("● starting mactop…", fg(Role::Mid))),
            Some(s) => {
                let mut t = String::new();
                if self.paused {
                    t.push_str("paused  ");
                }
                if self.source.is_replay() {
                    t.push_str("replay  ");
                }
                t.push_str(&s.clock);
                Line::from(Span::styled(t, fg(Role::Dim)))
            }
        };
        Line::from(left).render(area, buf);
        right.right_aligned().render(area, buf);
    }

    fn legend(&self, buf: &mut Buffer, area: Rect) {
        let state = if self.columns.on {
            "ratty: on"
        } else if self.ratty_found {
            "ratty: off"
        } else {
            "ratty: off (not detected)"
        };
        let keys = "←/→ rotate · ↑/↓ tilt · space pause · a auto-spin · r ratty · q quit";
        let room = (area.width as usize).saturating_sub(state.chars().count() + 3);
        let keys: String = if keys.chars().count() > room {
            keys.chars()
                .take(room.saturating_sub(1))
                .chain(['…'])
                .collect()
        } else {
            keys.into()
        };
        Line::from(Span::styled(format!(" {keys}"), fg(Role::Dim))).render(area, buf);
        Line::from(Span::styled(format!("{state} "), fg(Role::Dim)))
            .right_aligned()
            .render(area, buf);
    }
}

/// Chart cells, then core labels and scale ticks in dim wherever no column is drawn.
fn draw_chart(buf: &mut Buffer, area: Rect, chart: &scene::Chart, cores: &[scene::Core]) {
    for (y, row) in chart.cells.iter().enumerate() {
        for (x, &(ch, role)) in row.iter().enumerate() {
            if let Some(cell) = buf.cell_mut((area.x + x as u16, area.y + y as u16)) {
                cell.set_char(ch);
                if let Some(r) = role {
                    cell.set_style(fg(r));
                }
            }
        }
    }
    let text = |buf: &mut Buffer, r: i64, c: i64, s: &str| {
        for (j, ch) in s.chars().enumerate() {
            let (y, x) = (r, c + j as i64);
            if y < 0 || x < 0 || y >= area.height as i64 || x >= area.width as i64 {
                continue;
            }
            if matches!(
                chart.cells[y as usize][x as usize].1,
                None | Some(Role::Dim)
            ) && let Some(cell) = buf.cell_mut((area.x + x as u16, area.y + y as u16))
            {
                cell.set_char(ch).set_style(fg(Role::Dim));
            }
        }
    };
    for &(r, c, t) in &chart.ticks {
        text(buf, r, c, t);
    }
    for (&(r, c), core) in chart.labels.iter().zip(cores) {
        text(buf, r, c, &core.label);
    }
}
