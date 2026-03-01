use std::{
    process::{Command, Stdio},
    thread,
    time::{Duration, Instant},
};

use anyhow::{Context, Result};

#[derive(Debug, Clone)]
pub struct ShellResult {
    pub exit_code: Option<i32>,
    pub stdout: String,
    pub stderr: String,
    pub timed_out: bool,
    pub duration_ms: u128,
}

pub fn run_shell(command: &str, timeout: Duration, max_output_bytes: usize) -> Result<ShellResult> {
    let start = Instant::now();
    let mut child = Command::new("sh")
        .arg("-lc")
        .arg(command)
        .stdout(Stdio::piped())
        .stderr(Stdio::piped())
        .spawn()
        .with_context(|| format!("Failed to spawn shell command: {command}"))?;

    let mut timed_out = false;
    loop {
        if child.try_wait()?.is_some() {
            break;
        }

        if start.elapsed() >= timeout {
            timed_out = true;
            child.kill().context("Failed to kill timed out process")?;
            break;
        }

        thread::sleep(Duration::from_millis(50));
    }

    let output = child
        .wait_with_output()
        .context("Failed to collect shell output")?;

    let duration_ms = start.elapsed().as_millis();
    let stdout = String::from_utf8_lossy(&output.stdout).into_owned();
    let stderr = String::from_utf8_lossy(&output.stderr).into_owned();

    Ok(ShellResult {
        exit_code: output.status.code(),
        stdout: truncate_text(&stdout, max_output_bytes),
        stderr: truncate_text(&stderr, max_output_bytes),
        timed_out,
        duration_ms,
    })
}

fn truncate_text(input: &str, max_bytes: usize) -> String {
    if input.len() <= max_bytes {
        return input.to_string();
    }

    let marker = "\n...[truncated]";

    // When the limit is smaller than the marker itself, emit a
    // best-effort truncated marker that fits within max_bytes.
    if max_bytes < marker.len() {
        let mut out = String::new();
        for ch in marker.chars() {
            if out.len() + ch.len_utf8() > max_bytes {
                break;
            }
            out.push(ch);
        }
        return out;
    }

    let mut out = String::new();
    for ch in input.chars() {
        if out.len() + ch.len_utf8() + marker.len() > max_bytes {
            break;
        }
        out.push(ch);
    }
    out.push_str(marker);
    out
}

#[cfg(test)]
mod tests {
    use super::truncate_text;

    #[test]
    fn no_truncation_when_within_limit() {
        assert_eq!(truncate_text("abcdef", 6), "abcdef");
        assert_eq!(truncate_text("abcdef", 100), "abcdef");
    }

    #[test]
    fn truncates_with_marker_when_over_limit() {
        // "abcdefghijklmnopqrst" (20 bytes) with max_bytes=19 forces truncation.
        // Marker "\n...[truncated]" is 15 bytes, leaving 4 bytes for content.
        let output = truncate_text("abcdefghijklmnopqrst", 19);
        assert_eq!(output, "abcd\n...[truncated]");
    }

    #[test]
    fn truncates_to_just_marker_when_limit_smaller_than_marker() {
        // max_bytes=4 is smaller than the 15-byte marker, so output is a
        // best-effort truncation of the marker itself that fits in 4 bytes.
        let output = truncate_text("abcdef", 4);
        assert_eq!(output, "\n...");
        assert!(output.len() <= 4);
    }
}
