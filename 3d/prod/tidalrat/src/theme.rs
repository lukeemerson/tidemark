//! Monitor TUI colour roles (design-system/tokens.json) as ANSI slots, never 24-bit colour,
//! so the terminal's own palette decides the hue.

use ratatui::style::{Color, Modifier, Style};

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum Role {
    Text,
    Dim,
    Low,
    Mid,
    High,
    /// series-power (cyan): power, upload, and "▶ replay"
    Power,
}

/// `level()` from bin/monitor: under 50 low, 50–79 mid, 80 and up high.
pub fn level(p: f64) -> Role {
    if p >= 80.0 {
        Role::High
    } else if p >= 50.0 {
        Role::Mid
    } else {
        Role::Low
    }
}

pub fn fg(role: Role) -> Style {
    let c = match role {
        Role::Text => Color::Reset,
        Role::Dim => Color::DarkGray, // ANSI 8, ESC[90m
        Role::Low => Color::Green,
        Role::Mid => Color::Yellow,
        Role::High => Color::Red,
        Role::Power => Color::Cyan,
    };
    Style::new().fg(c)
}

pub fn bold(role: Role) -> Style {
    fg(role).add_modifier(Modifier::BOLD)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn level_thresholds() {
        assert_eq!(level(49.9), Role::Low);
        assert_eq!(level(50.0), Role::Mid);
        assert_eq!(level(79.9), Role::Mid);
        assert_eq!(level(80.0), Role::High);
    }
}
