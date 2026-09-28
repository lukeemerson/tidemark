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
    };
    Style::new().fg(c)
}

pub fn bold(role: Role) -> Style {
    fg(role).add_modifier(Modifier::BOLD)
}

/// Ratty draws true-colour meshes, so the level roles need hex there. These are the Alacritty
/// palette values from tokens.json (the yellow slot renders lilac).
pub fn hex(role: Role) -> [u8; 3] {
    match role {
        Role::High => [0xf0, 0x64, 0x59],
        Role::Mid => [0xc8, 0x98, 0xca],
        _ => [0xa3, 0xbe, 0x8c],
    }
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
