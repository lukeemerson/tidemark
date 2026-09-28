//! tidalrat: 3D views of the tidemark monitor data: `--cores` and `--terrain`.

mod alerts;
mod cores;
mod scene;
mod source;
mod terrain;
mod theme;
mod ui;

use std::process::ExitCode;

const USAGE: &str = "\
tidalrat — 3D views of your Mac's load

usage:
  tidalrat --cores   [--replay | --play <file>]
  tidalrat --terrain [--replay | --play <file>]

commands:
  --cores     per-core load as a rotating 3D bar chart
  --terrain   per-core load over time as a landscape you can scrub

options:
  --replay       play the bundled 90-second recording instead of live mactop
  --play <file>  play a saved `mactop --headless` stream at its recorded spacing
  -h, --help     show this help
";

#[derive(Clone, Copy, Debug, PartialEq)]
enum Command {
    Cores,
    Terrain,
}

#[derive(Debug, Default)]
pub struct Opts {
    pub replay: bool,
    pub play: Option<String>,
    pub frames: Option<u64>,
}

fn main() -> ExitCode {
    let args: Vec<String> = std::env::args().skip(1).collect();
    match parse(&args) {
        Ok(Some((cmd, opts))) => {
            let r = match cmd {
                Command::Cores => cores::run(&opts),
                Command::Terrain => terrain::run(&opts),
            };
            match r {
                Ok(()) => ExitCode::SUCCESS,
                Err(e) => {
                    eprintln!("tidalrat: {e}");
                    ExitCode::FAILURE
                }
            }
        }
        Ok(None) => {
            print!("{USAGE}");
            ExitCode::SUCCESS
        }
        Err(msg) => {
            eprint!("tidalrat: {msg}\n\n{USAGE}");
            ExitCode::from(2)
        }
    }
}

/// `Ok(None)` means print help.
fn parse(args: &[String]) -> Result<Option<(Command, Opts)>, String> {
    let mut opts = Opts::default();
    let mut command = None;
    let mut it = args.iter();
    while let Some(a) = it.next() {
        let set = |c: Command, cur: &mut Option<Command>| match cur {
            Some(_) => Err("give one command".to_string()),
            None => {
                *cur = Some(c);
                Ok(())
            }
        };
        match a.as_str() {
            "--cores" => set(Command::Cores, &mut command)?,
            "--terrain" => set(Command::Terrain, &mut command)?,
            "--replay" => opts.replay = true,
            "--play" => opts.play = Some(it.next().ok_or("--play needs a file")?.clone()),
            "-h" | "--help" => return Ok(None),
            // hidden: exit after N frames, for tests and captures
            "--frames" => {
                opts.frames = Some(
                    it.next()
                        .and_then(|n| n.parse().ok())
                        .ok_or("--frames needs a number")?,
                )
            }
            other => return Err(format!("unknown option {other}")),
        }
    }
    let Some(cmd) = command else {
        return if args.is_empty() {
            Ok(None)
        } else {
            Err("no command given".into())
        };
    };
    if opts.replay && opts.play.is_some() {
        return Err("use --replay or --play, not both".into());
    }
    Ok(Some((cmd, opts)))
}

#[cfg(test)]
mod tests {
    use super::*;

    fn p(s: &str) -> Result<Option<(Command, Opts)>, String> {
        parse(&s.split_whitespace().map(String::from).collect::<Vec<_>>())
    }

    #[test]
    fn parses_commands_and_flags() {
        let (c, o) = p("--cores --replay --frames 5").unwrap().unwrap();
        assert_eq!(c, Command::Cores);
        assert!(o.replay);
        assert_eq!(o.frames, Some(5));
        let (c, o) = p("--terrain --play rec.raw").unwrap().unwrap();
        assert_eq!(c, Command::Terrain);
        assert_eq!(o.play.as_deref(), Some("rec.raw"));
        assert!(p("").unwrap().is_none());
        assert!(p("--help").unwrap().is_none());
        assert!(p("--replay").is_err());
        assert!(p("--cores --bogus").is_err());
        assert!(p("--cores --terrain").is_err());
        assert!(p("--cores --ratty").is_err());
        assert!(p("--terrain --replay --play x").is_err());
        assert!(p("--terrain --play").is_err());
    }
}
