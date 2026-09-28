//! Samples from `mactop --headless` (live), the 90 recorded samples in
//! monitor-3d/shared/samples.json (replay, 1 Hz, looping), or a saved raw mactop stream
//! (`--play <file>`, at the recorded spacing).

use serde::Deserialize;
use serde_json::Value;
use std::io::{self, BufRead, BufReader};
use std::process::{Child, Command, Stdio};
use std::sync::mpsc::{self, Receiver, TryRecvError};
use std::time::{Duration, Instant};

#[derive(Clone, Debug, Default, PartialEq)]
pub struct Sample {
    pub cpu: f64,
    pub cores: Vec<f64>,
    /// HH:MM:SS from the sample's own timestamp
    pub clock: String,
    /// YYYY-MM-DD HH:MM, for "recorded …"
    pub recorded: String,
    /// seconds since 1970 (UTC) from the timestamp, if it parsed
    pub t: Option<f64>,
}

#[derive(Clone, Debug, PartialEq)]
pub struct Meta {
    pub name: String,
    pub e: usize,
    pub p: usize,
    pub gpu_cores: Option<usize>,
}

pub enum Source {
    Live {
        rx: Receiver<(Sample, Meta)>,
        child: Child,
        ended: bool,
    },
    Replay {
        samples: Vec<Sample>,
        i: usize,
        next: Instant,
    },
    Play {
        samples: Vec<(Sample, Meta)>,
        i: usize,
        start: Instant,
    },
}

const RECORDING: &str = include_str!("../../shared/samples.json");

#[derive(Deserialize)]
struct Recording {
    meta: RecMeta,
    samples: Vec<RecSample>,
}
#[derive(Deserialize)]
struct RecMeta {
    name: String,
    e: usize,
    p: usize,
    #[serde(rename = "gpuCores")]
    gpu_cores: usize,
}
#[derive(Deserialize)]
struct RecSample {
    t: String,
    cpu: f64,
    cores: Vec<f64>,
}

fn clock(ts: &str) -> String {
    ts.get(11..19).unwrap_or("").to_string()
}

fn recorded(ts: &str) -> String {
    match (ts.get(0..10), ts.get(11..16)) {
        (Some(d), Some(hm)) => format!("{d} {hm}"),
        _ => String::new(),
    }
}

/// `2026-09-26T19:44:43-04:00` (fraction and offset optional) -> seconds since 1970 UTC.
pub fn epoch(ts: &str) -> Option<f64> {
    let n = |a: usize, b: usize| ts.get(a..b)?.parse::<i64>().ok();
    let (y, mo, d) = (n(0, 4)?, n(5, 7)?, n(8, 10)?);
    let (h, mi, sec) = (n(11, 13)?, n(14, 16)?, n(17, 19)?);
    let rest = ts.get(19..).unwrap_or("");
    let frac_end = rest.find(['Z', '+', '-']).unwrap_or(rest.len());
    let frac: f64 = rest[..frac_end]
        .parse::<f64>()
        .ok()
        .filter(|_| rest.starts_with('.'))
        .unwrap_or(0.0);
    let tz = &rest[frac_end..];
    let off = if tz.len() >= 6 && (tz.starts_with('+') || tz.starts_with('-')) {
        let sign = if tz.starts_with('-') { -1 } else { 1 };
        sign * (tz.get(1..3)?.parse::<i64>().ok()? * 3600 + tz.get(4..6)?.parse::<i64>().ok()? * 60)
    } else {
        0
    };
    // days from civil (Howard Hinnant's algorithm)
    let yy = if mo <= 2 { y - 1 } else { y };
    let era = yy.div_euclid(400);
    let yoe = yy - era * 400;
    let doy = (153 * (mo + if mo > 2 { -3 } else { 9 }) + 2) / 5 + d - 1;
    let doe = yoe * 365 + yoe / 4 - yoe / 100 + doy;
    let days = era * 146097 + doe - 719468;
    Some((days * 86400 + h * 3600 + mi * 60 + sec - off) as f64 + frac)
}

fn sample(cpu: f64, cores: Vec<f64>, ts: &str) -> Sample {
    Sample {
        cpu,
        cores,
        clock: clock(ts),
        recorded: recorded(ts),
        t: epoch(ts),
    }
}

pub fn recording() -> (Meta, Vec<Sample>) {
    let r: Recording = serde_json::from_str(RECORDING).expect("bundled samples.json");
    let meta = Meta {
        name: r.meta.name,
        e: r.meta.e,
        p: r.meta.p,
        gpu_cores: Some(r.meta.gpu_cores),
    };
    let s = r
        .samples
        .into_iter()
        .map(|s| sample(s.cpu, s.cores, &s.t))
        .collect();
    (meta, s)
}

/// One line of mactop's streamed JSON array: `[{...}` then `,{...}`.
pub fn parse_line(line: &str) -> Option<(Sample, Meta)> {
    let body = line.trim().trim_start_matches(['[', ',']);
    if !body.starts_with('{') {
        return None;
    }
    let v: Value = serde_json::from_str(body.trim_end_matches(']')).ok()?;
    let info = &v["system_info"];
    let n = |k: &str| info[k].as_u64().map(|x| x as usize);
    let meta = Meta {
        name: info["name"].as_str().unwrap_or("").to_string(),
        e: n("e_core_count")?,
        p: n("p_core_count")?,
        gpu_cores: n("gpu_core_count"),
    };
    let cores = v["core_usages"]
        .as_array()?
        .iter()
        .filter_map(Value::as_f64)
        .collect();
    let s = sample(
        v["cpu_usage"].as_f64()?,
        cores,
        v["timestamp"].as_str().unwrap_or(""),
    );
    Some((s, meta))
}

/// The source the options ask for: `--play <file>`, `--replay`, or live mactop (falling back
/// to the bundled recording if mactop won't start).
pub fn open(replay: bool, play: Option<&str>) -> io::Result<Source> {
    Ok(match play {
        Some(path) => Source::play(path)?,
        None if replay => Source::replay(),
        None => Source::live().unwrap_or_else(Source::replay),
    })
}

fn sysctl(key: &str) -> Option<String> {
    let out = Command::new("sysctl").args(["-n", key]).output().ok()?;
    Some(String::from_utf8_lossy(&out.stdout).trim().to_string()).filter(|s| !s.is_empty())
}

/// Machine facts available before mactop's first sample (bin/monitor does the same).
pub fn static_meta() -> Option<Meta> {
    Some(Meta {
        name: sysctl("machdep.cpu.brand_string")?,
        e: sysctl("hw.perflevel1.physicalcpu")?.parse().ok()?,
        p: sysctl("hw.perflevel0.physicalcpu")?.parse().ok()?,
        gpu_cores: None,
    })
}

impl Source {
    /// Live mactop, or `None` if it can't be started.
    pub fn live() -> Option<Source> {
        let mut child = Command::new("mactop")
            .args(["--headless", "--count", "0", "-i", "1000"])
            .stdin(Stdio::null())
            .stdout(Stdio::piped())
            .stderr(Stdio::null())
            .spawn()
            .ok()?;
        let out = child.stdout.take()?;
        let (tx, rx) = mpsc::channel();
        std::thread::spawn(move || {
            for line in BufReader::new(out).lines().map_while(Result::ok) {
                if let Some(s) = parse_line(&line)
                    && tx.send(s).is_err()
                {
                    break;
                }
            }
        });
        Some(Source::Live {
            rx,
            child,
            ended: false,
        })
    }

    pub fn replay() -> Source {
        Source::Replay {
            samples: recording().1,
            i: 0,
            next: Instant::now(),
        }
    }

    /// A saved raw `mactop --headless` stream, played back at its recorded spacing.
    pub fn play(path: &str) -> io::Result<Source> {
        let text = std::fs::read_to_string(path)?;
        let samples: Vec<_> = text.lines().filter_map(parse_line).collect();
        if samples.is_empty() {
            return Err(io::Error::new(
                io::ErrorKind::InvalidData,
                format!("{path}: no mactop samples in it"),
            ));
        }
        Ok(Source::Play {
            samples,
            i: 0,
            start: Instant::now(),
        })
    }

    pub fn is_replay(&self) -> bool {
        matches!(self, Source::Replay { .. } | Source::Play { .. })
    }

    /// For `--play`: (samples played, samples in the file).
    pub fn progress(&self) -> Option<(usize, usize)> {
        match self {
            Source::Play { samples, i, .. } => Some((*i, samples.len())),
            _ => None,
        }
    }

    /// True once live mactop has exited or closed its output.
    pub fn ended(&self) -> bool {
        matches!(self, Source::Live { ended: true, .. })
    }

    /// Every sample that arrived since the last call, oldest first. Live and played-back
    /// samples carry their meta.
    pub fn poll(&mut self) -> Vec<(Sample, Option<Meta>)> {
        let mut out = vec![];
        match self {
            Source::Live { rx, ended, .. } => loop {
                match rx.try_recv() {
                    Ok((s, m)) => out.push((s, Some(m))),
                    Err(TryRecvError::Empty) => break,
                    Err(TryRecvError::Disconnected) => {
                        *ended = true;
                        break;
                    }
                }
            },
            Source::Replay { samples, i, next } => {
                while Instant::now() >= *next {
                    *next += Duration::from_secs(1);
                    out.push((samples[*i].clone(), None));
                    *i = (*i + 1) % samples.len();
                }
            }
            Source::Play { samples, i, start } => {
                // sample k is due at its timestamp's offset from the first (1 s apart if a
                // timestamp is missing)
                let t0 = samples[0].0.t;
                let due = |k: usize, s: &Sample| match (s.t, t0) {
                    (Some(t), Some(t0)) => t - t0,
                    _ => k as f64,
                };
                let now = start.elapsed().as_secs_f64();
                while *i < samples.len() && due(*i, &samples[*i].0) <= now {
                    let (s, m) = samples[*i].clone();
                    out.push((s, Some(m)));
                    *i += 1;
                }
            }
        }
        out
    }
}

impl Drop for Source {
    fn drop(&mut self) {
        if let Source::Live { child, .. } = self {
            let _ = child.kill();
            let _ = child.wait();
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn recording_has_90_samples_of_10_cores() {
        let (meta, s) = recording();
        assert_eq!(s.len(), 90);
        assert_eq!((meta.e, meta.p), (4, 6));
        assert!(s.iter().all(|x| x.cores.len() == 10));
        assert_eq!(s[0].clock, "19:44:43");
    }

    #[test]
    fn parses_recorded_mactop_stream() {
        // copy of tidemark's fixture (internal/source/testdata): the raw `mactop --headless` stream
        let raw = include_str!("../testdata/mactop.raw");
        let parsed: Vec<_> = raw.lines().filter_map(parse_line).collect();
        assert_eq!(parsed.len(), 90);
        let (s, m) = &parsed[0];
        assert_eq!(m.name, "Apple M2 Pro");
        assert_eq!((m.e, m.p, m.gpu_cores), (4, 6, Some(16)));
        assert_eq!(s.cores.len(), 10);
        assert!((s.cpu - 28.07).abs() < 0.01);
        assert_eq!(s.clock, "19:44:43");
        assert_eq!(s.recorded, "2026-09-26 19:44");
    }

    #[test]
    fn epoch_parses_offsets_and_fractions() {
        // 2026-09-26T23:44:43Z
        let utc = 1_790_466_283.0;
        assert_eq!(epoch("2026-09-26T19:44:43-04:00"), Some(utc));
        assert_eq!(epoch("2026-09-26T23:44:43Z"), Some(utc));
        assert_eq!(epoch("2026-09-26T23:44:43.5Z"), Some(utc + 0.5));
        assert_eq!(epoch("1970-01-01T00:00:00Z"), Some(0.0));
        assert_eq!(epoch("garbage"), None);
    }

    #[test]
    fn play_emits_at_the_recorded_spacing() {
        let mut src = Source::play("testdata/mactop.raw").unwrap();
        assert_eq!(src.progress(), Some((0, 90)));
        let first = src.poll(); // the first sample is due at once
        assert_eq!(first.len(), 1);
        assert!(src.poll().is_empty()); // the next is about a second later
        if let Source::Play { start, .. } = &mut src {
            *start -= Duration::from_secs(3600); // jump past the end
        }
        assert_eq!(src.poll().len(), 89);
        assert_eq!(src.progress(), Some((90, 90)));
    }

    #[test]
    fn ignores_non_object_lines() {
        assert!(parse_line("]").is_none());
        assert!(parse_line("").is_none());
    }
}
