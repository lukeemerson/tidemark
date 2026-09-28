//! tidalrat: 3D views of the tidemark monitor data. `tidalrat --cores` is the first.

mod cores;
mod ratty;
mod scene;
mod source;
mod theme;

use std::process::ExitCode;

const USAGE: &str = "\
tidalrat — 3D views of your Mac's load

usage:
  tidalrat --cores [--replay] [--ratty]

commands:
  --cores     per-core load as a rotating 3D bar chart (live mactop)

options:
  --replay    play the bundled 90-second recording instead of live mactop
  --ratty     draw the columns as real 3D objects (auto when running inside Ratty)
  -h, --help  show this help
";

fn main() -> ExitCode {
    let args: Vec<String> = std::env::args().skip(1).collect();
    match parse(&args) {
        Ok(Some(opts)) => match cores::run(opts) {
            Ok(()) => ExitCode::SUCCESS,
            Err(e) => {
                eprintln!("tidalrat: {e}");
                ExitCode::FAILURE
            }
        },
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
fn parse(args: &[String]) -> Result<Option<cores::Opts>, String> {
    let mut opts = cores::Opts {
        replay: false,
        ratty: false,
        frames: None,
    };
    let mut command = false;
    let mut it = args.iter();
    while let Some(a) = it.next() {
        match a.as_str() {
            "--cores" => command = true,
            "--replay" => opts.replay = true,
            "--ratty" => opts.ratty = true,
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
    if !command {
        return if args.is_empty() {
            Ok(None)
        } else {
            Err("no command given".into())
        };
    }
    Ok(Some(opts))
}

#[cfg(test)]
mod tests {
    use super::*;

    fn p(s: &str) -> Result<Option<cores::Opts>, String> {
        parse(&s.split_whitespace().map(String::from).collect::<Vec<_>>())
    }

    #[test]
    fn parses_commands_and_flags() {
        let o = p("--cores --replay --frames 5").unwrap().unwrap();
        assert!(o.replay && !o.ratty);
        assert_eq!(o.frames, Some(5));
        assert!(p("").unwrap().is_none());
        assert!(p("--help").unwrap().is_none());
        assert!(p("--replay").is_err());
        assert!(p("--cores --bogus").is_err());
    }
}
