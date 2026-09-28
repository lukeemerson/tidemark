//! Samples from `mactop --headless` (live) or the 90 recorded samples in
//! monitor-3d/shared/samples.json (replay, 1 Hz, looping).

use serde::Deserialize;
use serde_json::Value;
use std::io::{BufRead, BufReader};
use std::process::{Child, Command, Stdio};
use std::sync::mpsc::{self, Receiver};
use std::time::{Duration, Instant};

#[derive(Clone, Debug, Default, PartialEq)]
pub struct Sample {
    pub cpu: f64,
    pub cores: Vec<f64>,
    /// HH:MM:SS from the sample's own timestamp
    pub clock: String,
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
    },
    Replay {
        samples: Vec<Sample>,
        i: usize,
        next: Instant,
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
        .map(|s| Sample {
            cpu: s.cpu,
            cores: s.cores,
            clock: clock(&s.t),
        })
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
    let sample = Sample {
        cpu: v["cpu_usage"].as_f64()?,
        cores,
        clock: clock(v["timestamp"].as_str().unwrap_or("")),
    };
    Some((sample, meta))
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
        Some(Source::Live { rx, child })
    }

    pub fn replay() -> Source {
        Source::Replay {
            samples: recording().1,
            i: 0,
            next: Instant::now(),
        }
    }

    pub fn is_replay(&self) -> bool {
        matches!(self, Source::Replay { .. })
    }

    /// The newest sample since the last call, if any. Live meta comes with each sample.
    pub fn poll(&mut self) -> Option<(Sample, Option<Meta>)> {
        match self {
            Source::Live { rx, .. } => rx.try_iter().last().map(|(s, m)| (s, Some(m))),
            Source::Replay { samples, i, next } => {
                if Instant::now() < *next {
                    return None;
                }
                *next += Duration::from_secs(1);
                let s = samples[*i].clone();
                *i = (*i + 1) % samples.len();
                Some((s, None))
            }
        }
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
        // tidemark's own test fixture: the raw `mactop --headless` stream
        let raw = include_str!("../../../internal/source/testdata/mactop.raw");
        let parsed: Vec<_> = raw.lines().filter_map(parse_line).collect();
        assert_eq!(parsed.len(), 90);
        let (s, m) = &parsed[0];
        assert_eq!(m.name, "Apple M2 Pro");
        assert_eq!((m.e, m.p, m.gpu_cores), (4, 6, Some(16)));
        assert_eq!(s.cores.len(), 10);
        assert!((s.cpu - 28.07).abs() < 0.01);
        assert_eq!(s.clock, "19:44:43");
    }

    #[test]
    fn ignores_non_object_lines() {
        assert!(parse_line("]").is_none());
        assert!(parse_line("").is_none());
    }
}
