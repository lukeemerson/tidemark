//! `tidalrat --terrain`: per-core load over time as a braille landscape, with a scrub cursor
//! over the last 400 samples (prototype C in the terminal).

use crate::Opts;
use crate::scene::{self, Terrain};
use crate::source::{self, Meta, Sample, Source};
use crate::theme::{Role, bold, fg, level};
use crate::ui;
use ratatui::{
    DefaultTerminal, Frame,
    buffer::Buffer,
    crossterm::event::{self, Event, KeyCode, KeyEventKind, KeyModifiers},
    layout::Rect,
    text::{Line, Span},
    widgets::{Paragraph, Widget},
};
use std::collections::VecDeque;
use std::io;
use std::time::{Duration, Instant};

const FRAME: Duration = Duration::from_millis(50);
const MIN_W: u16 = 60;
const MIN_H: u16 = 20;
/// samples kept for scrubbing (tidemark's histLen)
const HISTORY: usize = 400;
/// samples shown in the landscape, ending at the cursor
const WINDOW: usize = 90;
/// cells in the scrub track
const TRACK: usize = 30;

struct App {
    source: Source,
    meta: Option<Meta>,
    hist: VecDeque<Sample>,
    /// when each sample in `hist` arrived, in seconds since `clock`; ages come from these,
    /// since recorded timestamps jump when the recording loops or replaces live data
    arrived: VecDeque<f64>,
    clock: Instant,
    /// samples back from the newest; 0 = live edge
    offset: usize,
    paused: bool,
    yaw: f64,
    pitch: f64,
    spin: bool,
    mactop_stopped: bool,
}

pub fn run(opts: &Opts) -> io::Result<()> {
    let source = source::open(opts.replay, opts.play.as_deref())?;
    let meta = if source.is_replay() {
        opts.play.is_none().then(|| source::recording().0)
    } else {
        source::static_meta()
    };
    let mut app = App {
        source,
        meta,
        hist: VecDeque::new(),
        arrived: VecDeque::new(),
        clock: Instant::now(),
        offset: 0,
        paused: false,
        yaw: 0.6,
        pitch: 0.45,
        spin: false,
        mactop_stopped: false,
    };
    let mut terminal = ratatui::try_init()?;
    let result = app.run(&mut terminal, opts.frames);
    ratatui::restore();
    result
}

impl App {
    fn run(&mut self, terminal: &mut DefaultTerminal, frames: Option<u64>) -> io::Result<()> {
        let (mut last, mut drawn) = (Instant::now(), 0u64);
        loop {
            let dt = last.elapsed().as_secs_f64();
            last = Instant::now();
            for (s, m) in self.source.poll() {
                if let Some(m) = m {
                    self.meta = Some(m);
                }
                self.push(s);
            }
            if self.source.ended() {
                // mactop exited: say so, and keep the screen alive on the recording, starting
                // a fresh history so live rows don't sit under the recording's core layout
                self.source = Source::replay();
                self.mactop_stopped = true;
                self.meta = Some(source::recording().0);
                self.hist.clear();
                self.arrived.clear();
                self.offset = 0;
            }
            if self.spin {
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
                if let Event::Key(k) = event::read()?
                    && k.kind == KeyEventKind::Press
                {
                    match k.code {
                        KeyCode::Char('q') | KeyCode::Esc => return Ok(()),
                        KeyCode::Char('c') if k.modifiers.contains(KeyModifiers::CONTROL) => {
                            return Ok(());
                        }
                        KeyCode::Left => self.yaw -= 0.15,
                        KeyCode::Right => self.yaw += 0.15,
                        KeyCode::Up => self.pitch = (self.pitch + 0.08).min(1.3),
                        KeyCode::Down => self.pitch = (self.pitch - 0.08).max(0.1),
                        KeyCode::Char(' ') => {
                            // resuming goes back to the live edge
                            self.paused = !self.paused;
                            if !self.paused {
                                self.offset = 0;
                            }
                        }
                        KeyCode::Char('[') => self.scrub(1),
                        KeyCode::Char(']') => self.scrub(-1),
                        KeyCode::Char('{') => self.scrub(30),
                        KeyCode::Char('}') => self.scrub(-30),
                        KeyCode::Char('a') => self.spin = !self.spin,
                        _ => {}
                    }
                }
            }
        }
    }

    /// Add a sample. While paused the cursor stays on the sample it was on.
    fn push(&mut self, s: Sample) {
        let at = self.clock.elapsed().as_secs_f64();
        self.push_at(s, at);
    }

    fn push_at(&mut self, s: Sample, at: f64) {
        self.hist.push_back(s);
        self.arrived.push_back(at);
        if self.paused {
            self.offset += 1;
        }
        if self.hist.len() > HISTORY {
            self.hist.pop_front();
            self.arrived.pop_front();
        }
        self.offset = self.offset.min(self.hist.len().saturating_sub(1));
    }

    /// Move the cursor `back` samples into the past (negative = toward now); pauses.
    fn scrub(&mut self, back: i64) {
        self.paused = true;
        let max = self.hist.len().saturating_sub(1) as i64;
        self.offset = (self.offset as i64 + back).clamp(0, max) as usize;
    }

    fn cursor(&self) -> Option<(usize, &Sample)> {
        let i = self.hist.len().checked_sub(1 + self.offset)?;
        Some((i, &self.hist[i]))
    }

    /// Seconds between sample `i` arriving and the newest sample arriving.
    fn age(&self, i: usize) -> i64 {
        let newest = self.arrived.len() - 1;
        (self.arrived[newest] - self.arrived[i]).round() as i64
    }

    fn draw(&mut self, f: &mut Frame) {
        let full = f.area();
        if full.width < MIN_W || full.height < MIN_H {
            let msg = format!("resize to at least {MIN_W}×{MIN_H}");
            Paragraph::new(Span::styled(msg, fg(Role::Dim)))
                .centered()
                .render(
                    Rect {
                        y: full.height / 2,
                        height: 1,
                        ..full
                    },
                    f.buffer_mut(),
                );
            return;
        }
        // one blank column each side (design-system cell-xpad)
        let (x0, w, h) = (1, full.width - 2, full.height);
        self.header(f.buffer_mut(), Rect::new(x0, 0, w, 1));

        let cursor = self.cursor();
        let right = match cursor {
            Some((_, s)) => Span::styled(format!(" {:.0}% ", s.cpu), bold(level(s.cpu))),
            None => Span::styled(" — ", fg(Role::Dim)),
        };
        let block = ui::frame("terrain · 3d", right);
        let frame_area = Rect::new(x0, 1, w, h - 2);
        let inner = block.inner(frame_area);
        block.render(frame_area, f.buffer_mut());
        let chart_area = Rect {
            x: inner.x + 1,
            width: inner.width.saturating_sub(2),
            ..inner
        };

        let (rows, back_label) = match cursor {
            Some((i, _)) => {
                let start = (i + 1).saturating_sub(WINDOW);
                let rows: Vec<Vec<f64>> = (start..=i).map(|j| self.hist[j].cores.clone()).collect();
                let label = if rows.len() > 1 {
                    format!("t−{}s", self.age(start))
                } else {
                    String::new()
                };
                (rows, label)
            }
            None => (vec![], String::new()),
        };
        let names: Vec<String> = match &self.meta {
            Some(m) => scene::core_layout(m.e, m.p)
                .into_iter()
                .map(|c| c.label)
                .collect(),
            None => {
                let n = rows.last().map_or(0, Vec::len);
                (1..=n).map(|i| format!("C{i}")).collect()
            }
        };
        let terrain = Terrain {
            lanes: names.len().max(1),
            rows: &rows,
            window: WINDOW,
            back_label,
        };
        let cam = terrain.camera(self.yaw, self.pitch);
        let chart = terrain.render(chart_area.width as usize, chart_area.height as usize, &cam);
        ui::draw_chart(f.buffer_mut(), chart_area, &chart, &names);

        self.legend(f.buffer_mut(), Rect::new(x0, h - 1, w, 1));
    }

    fn track(&self, filled_of: (usize, usize)) -> Vec<Span<'static>> {
        let (a, n) = filled_of;
        let f = if n == 0 {
            0
        } else {
            (TRACK * a).div_ceil(n).min(TRACK)
        };
        vec![
            Span::styled("◀ ", fg(Role::Dim)),
            Span::styled("▮".repeat(f), fg(Role::Mid)),
            Span::styled("▯".repeat(TRACK - f), fg(Role::Dim)),
            Span::styled(" ▶", fg(Role::Dim)),
        ]
    }

    /// Right side of the header, richest first; the caller picks the first that fits.
    fn status(&self) -> Vec<Vec<Span<'static>>> {
        let Some((i, s)) = self.cursor() else {
            return vec![vec![Span::styled("● starting mactop…", fg(Role::Mid))]];
        };
        let stopped = self
            .mactop_stopped
            .then(|| Span::styled("mactop stopped  ", fg(Role::Mid)));
        if self.paused {
            let mut base = vec![Span::styled("‖ paused   ", fg(Role::Mid))];
            base.extend(self.track((i + 1, self.hist.len())));
            base.push(Span::styled(format!("  t−{}s", self.age(i)), fg(Role::Dim)));
            let mut full = base.clone();
            full.push(Span::styled(
                "   [ ] step  { } ±30  space live",
                fg(Role::Dim),
            ));
            return vec![full, base];
        }
        let mut replay = vec![];
        replay.extend(stopped.clone());
        if self.source.is_replay() {
            replay.push(Span::styled("▶ replay", fg(Role::Power)));
            let pos = match self.source.progress() {
                Some((a, n)) => format!(" {a}/{n}"),
                None => String::new(),
            };
            replay.push(Span::styled(
                format!("{pos} · 1× · recorded {}", s.recorded),
                fg(Role::Dim),
            ));
            let mut with_track = replay.clone();
            if let Some(p) = self.source.progress() {
                with_track.push(Span::raw("   "));
                with_track.extend(self.track(p));
            }
            return vec![with_track, replay];
        }
        vec![vec![Span::styled(s.clock.clone(), fg(Role::Dim))]]
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
        let options = self.status();
        let fits = |l: &Line| l.width() + 13 <= area.width as usize; // keep room for the name
        let right = options
            .iter()
            .map(|o| Line::from(o.clone()))
            .find(fits)
            .unwrap_or_else(|| Line::from(options.last().cloned().unwrap_or_default()));
        // the left side gets what the right leaves, minus a one-cell gap; if the core counts
        // don't fit whole, show just the chip name (as design's paused mockup does)
        let room = area.width.saturating_sub(right.width() as u16 + 1);
        if Line::from(left.clone()).width() > room as usize {
            left.truncate(1);
        }
        Line::from(left).render(
            Rect {
                width: room,
                ..area
            },
            buf,
        );
        right.right_aligned().render(area, buf);
    }

    fn legend(&self, buf: &mut Buffer, area: Rect) {
        let state = format!("history {}/{HISTORY}", self.hist.len());
        let keys = "←/→ rotate · ↑/↓ tilt · space pause · [ ] step · { } ±30 · a spin · q quit";
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

#[cfg(test)]
mod tests {
    use super::*;

    fn app(n: usize) -> App {
        let mut a = App {
            source: Source::replay(),
            meta: None,
            hist: VecDeque::new(),
            arrived: VecDeque::new(),
            clock: Instant::now(),
            offset: 0,
            paused: false,
            yaw: 0.0,
            pitch: 0.45,
            spin: false,
            mactop_stopped: false,
        };
        for (i, s) in source::recording()
            .1
            .into_iter()
            .cycle()
            .take(n)
            .enumerate()
        {
            let mut s = s;
            s.cpu = i as f64; // tag each sample with its arrival index
            a.push_at(s, i as f64 * 1.5); // arriving 1.5 s apart
        }
        a
    }

    #[test]
    fn age_is_arrival_time_even_when_recorded_timestamps_loop() {
        // 100 samples: the 90-sample recording loops, so recorded time jumps back at 90
        let a = app(100);
        assert_eq!(a.age(99), 0);
        assert_eq!(a.age(0), 149); // 99 × 1.5 s, rounded
        assert_eq!(a.age(95), 6);
    }

    #[test]
    fn history_is_bounded_at_400() {
        let a = app(450);
        assert_eq!(a.hist.len(), HISTORY);
        assert_eq!(a.cursor().unwrap().1.cpu, 449.0);
    }

    #[test]
    fn scrub_steps_pauses_and_clamps() {
        let mut a = app(100);
        a.scrub(1);
        assert!(a.paused);
        assert_eq!(a.cursor().unwrap().1.cpu, 98.0);
        a.scrub(30);
        assert_eq!(a.cursor().unwrap().1.cpu, 68.0);
        a.scrub(-30);
        a.scrub(-30);
        assert_eq!(a.offset, 0);
        a.scrub(1000);
        assert_eq!(a.cursor().unwrap().1.cpu, 0.0);
    }

    #[test]
    fn paused_cursor_stays_on_its_sample_while_data_keeps_arriving() {
        let mut a = app(100);
        a.scrub(10);
        let held = a.cursor().unwrap().1.cpu;
        let next = a.hist[0].clone();
        a.push(next.clone());
        a.push(next);
        assert_eq!(a.cursor().unwrap().1.cpu, held);
        assert_eq!(a.hist.len(), 102);
    }
}
