//! Drawing shared by the tidalrat commands: the heavy Monitor TUI frame and chart cells.

use crate::scene;
use crate::theme::{Role, bold, fg};
use ratatui::{
    buffer::Buffer,
    layout::Rect,
    text::{Line, Span},
    widgets::{Block, BorderType},
};

/// The heavy frame: `┏━ title ━━━ right ━┓` in dim, the title bold.
pub fn frame<'a>(title: &'a str, right: Span<'a>) -> Block<'a> {
    Block::bordered()
        .border_type(BorderType::Thick)
        .border_style(fg(Role::Dim))
        .title(Line::from(vec![
            Span::styled("━", fg(Role::Dim)),
            Span::styled(format!(" {title} "), bold(Role::Text)),
        ]))
        .title(Line::from(vec![right, Span::styled("━", fg(Role::Dim))]).right_aligned())
}

/// Chart cells, scale ticks where they fit whole, then core labels on top of everything.
pub fn draw_chart(buf: &mut Buffer, area: Rect, chart: &scene::Chart, labels: &[String]) {
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
    let free = |y: i64, x: i64| {
        y >= 0
            && x >= 0
            && y < area.height as i64
            && x < area.width as i64
            && matches!(
                chart.cells[y as usize][x as usize].1,
                None | Some(Role::Dim)
            )
    };
    let text = |buf: &mut Buffer, r: i64, c: i64, s: &str| {
        for (j, ch) in s.chars().enumerate() {
            let x = c + j as i64;
            if free(r, x)
                && let Some(cell) = buf.cell_mut((area.x + x as u16, area.y + r as u16))
            {
                cell.set_char(ch).set_style(fg(Role::Dim));
            }
        }
    };
    // a tick label goes left of its anchor, else right of it, and only where all of it fits,
    // so "100%" never shows as "0%"
    for (r, post, t) in &chart.ticks {
        let (r, post) = (*r, *post);
        let n = t.chars().count() as i64;
        if let Some(c) = [post - n - 1, post + 2]
            .into_iter()
            .find(|&c| (c..c + n).all(|x| free(r, x)))
        {
            text(buf, r, c, t);
        }
    }
    // labels always show, over the chart, so every core stays named
    for (&(r, c), label) in chart.labels.iter().zip(labels) {
        for (j, ch) in label.chars().enumerate() {
            let x = c + j as i64;
            if r >= 0
                && x >= 0
                && r < area.height as i64
                && x < area.width as i64
                && let Some(cell) = buf.cell_mut((area.x + x as u16, area.y + r as u16))
            {
                cell.set_char(ch).set_style(fg(Role::Dim));
            }
        }
    }
}
