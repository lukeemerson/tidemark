//! The three diagnosis rules from SPEC.md §3, ported from tidemark's internal/ui/diagnose.go
//! with the same thresholds, priority and clearing. tidalrat only needs to know when a rule
//! starts firing (an alert), not the rule text.

use crate::source::Sample;
use std::collections::VecDeque;

const CLEAR_AFTER: u32 = 5; // samples below threshold before a rule clears
const SWAP_WINDOW: usize = 10; // samples the swap rise is measured over
const SWAP_RISE: f64 = 256.0 * 1024.0 * 1024.0; // bytes of swap growth that counts as a storm
const RUNAWAY_CPU: f64 = 100.0; // one full core, in mactop's per-core CPU %
const RUNAWAY_RUNS: u32 = 10; // samples the same top process must stay at or above it

const THROTTLE: usize = 0;
const SWAP: usize = 1;
const RUNAWAY: usize = 2;

/// Each rule's state from sample to sample.
#[derive(Default)]
pub struct Rules {
    on: [bool; 3],
    quiet: [u32; 3],
    swaps: VecDeque<f64>,
    run_pid: i64,
    run_n: u32,
}

impl Rules {
    /// Fold one sample in. `pressure` is `kern.memorystatus_vm_pressure_level` when live, `None`
    /// on replay (so the swap rule can't fire, as in tidemark's -play). Returns true when any
    /// rule starts firing on this sample.
    pub fn step(&mut self, s: &Sample, pressure: Option<i64>) -> bool {
        let mut hit = [false; 3];
        hit[THROTTLE] = !s.thermal.is_empty() && s.thermal != "Nominal";

        self.swaps.push_back(s.swap_used);
        if self.swaps.len() > SWAP_WINDOW + 1 {
            self.swaps.pop_front();
        }
        let rise = self.swaps.back().unwrap_or(&0.0) - self.swaps.front().unwrap_or(&0.0);
        hit[SWAP] = pressure.is_some_and(|p| p >= 2)
            && self.swaps.len() == SWAP_WINDOW + 1
            && rise > SWAP_RISE;

        match s.top {
            Some((pid, cpu)) if cpu >= RUNAWAY_CPU => {
                if pid == self.run_pid {
                    self.run_n += 1;
                } else {
                    (self.run_pid, self.run_n) = (pid, 1);
                }
                hit[RUNAWAY] = self.run_n >= RUNAWAY_RUNS;
            }
            _ => (self.run_pid, self.run_n) = (0, 0),
        }

        let mut fired = false;
        for ((hit, on), quiet) in hit.iter().zip(&mut self.on).zip(&mut self.quiet) {
            if *hit {
                fired |= !*on;
                (*on, *quiet) = (true, 0);
            } else if *on {
                *quiet += 1;
                if *quiet >= CLEAR_AFTER {
                    *on = false;
                }
            }
        }
        fired
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::source::{self, parse_line};

    fn fired_at(samples: &[Sample], pressure: Option<i64>) -> Vec<usize> {
        let mut r = Rules::default();
        samples
            .iter()
            .enumerate()
            .filter(|(_, s)| r.step(s, pressure))
            .map(|(i, _)| i)
            .collect()
    }

    #[test]
    fn runaway_fires_where_tidemark_does() {
        // tidemark's Go rules fire runaway at samples 10 and 56 of this recording
        let raw = include_str!("../testdata/mactop.raw");
        let samples: Vec<Sample> = raw.lines().filter_map(parse_line).map(|(s, _)| s).collect();
        assert_eq!(fired_at(&samples, None), vec![10, 56]);
        // the bundled --replay copy of the same recording agrees
        assert_eq!(fired_at(&source::recording().1, None), vec![10, 56]);
    }

    fn at(swap_gb: f64, thermal: &str) -> Sample {
        Sample {
            swap_used: swap_gb * 1073741824.0,
            thermal: thermal.into(),
            ..Default::default()
        }
    }

    #[test]
    fn swap_needs_pressure_and_a_full_window() {
        let rising: Vec<Sample> = (0..20).map(|i| at(i as f64 * 0.05, "Nominal")).collect();
        assert_eq!(fired_at(&rising, Some(2)), vec![10]); // +0.5 GB over 10 samples
        assert!(fired_at(&rising, Some(1)).is_empty()); // pressure normal
        assert!(fired_at(&rising, None).is_empty()); // replay: no pressure
    }

    #[test]
    fn throttle_fires_once_and_clears_after_five_quiet_samples() {
        let states = [
            "Nominal", "Fair", "Fair", "Nominal", "Nominal", "Nominal", "Nominal", "Nominal",
            "Serious",
        ];
        let s: Vec<Sample> = states.iter().map(|t| at(0.0, t)).collect();
        assert_eq!(fired_at(&s, None), vec![1, 8]);
        let short_gap = ["Fair", "Nominal", "Nominal", "Fair"];
        let s: Vec<Sample> = short_gap.iter().map(|t| at(0.0, t)).collect();
        assert_eq!(fired_at(&s, None), vec![0]); // back within 5 samples: still the same alert
    }
}
