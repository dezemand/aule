use std::{
    io::Read,
    os::unix::process::CommandExt as _,
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
        // Start the shell in its own process group so we can kill the
        // entire group (including any child processes) on timeout.
        .process_group(0)
        .spawn()
        .with_context(|| format!("Failed to spawn shell command: {command}"))?;

    let child_pid = child.id();

    let mut timed_out = false;
    loop {
        if child.try_wait()?.is_some() {
            break;
        }

        if start.elapsed() >= timeout {
            timed_out = true;
            // Kill the entire process group, falling back to child.kill()
            // if the group kill fails.
            // Safety: child_pid is a valid pid from a process we spawned.
            let pgid = child_pid as i32;
            if unsafe { libc::kill(-pgid, libc::SIGKILL) } != 0 {
                child.kill().context("Failed to kill timed out process")?;
            }
            break;
        }

        thread::sleep(Duration::from_millis(50));
    }

    // Read stdout/stderr with a cap so a runaway process can't OOM us.
    // We read cap+1 bytes to detect whether truncation occurred.
    let cap = max_output_bytes as u64;
    let (stdout, stdout_truncated) = read_capped(child.stdout.take(), cap)?;
    let (stderr, stderr_truncated) = read_capped(child.stderr.take(), cap)?;

    let status = child.wait().context("Failed to wait for child process")?;
    let duration_ms = start.elapsed().as_millis();

    let marker = "\n...[truncated]";

    Ok(ShellResult {
        exit_code: status.code(),
        stdout: if stdout_truncated {
            format!("{stdout}{marker}")
        } else {
            stdout
        },
        stderr: if stderr_truncated {
            format!("{stderr}{marker}")
        } else {
            stderr
        },
        timed_out,
        duration_ms,
    })
}

/// Read up to `cap` bytes from an optional pipe. Returns the string and
/// whether the output was truncated.
fn read_capped(pipe: Option<impl Read>, cap: u64) -> Result<(String, bool)> {
    let Some(pipe) = pipe else {
        return Ok((String::new(), false));
    };
    // Read cap+1 to detect if there's more data beyond the cap.
    let mut buf = Vec::new();
    pipe.take(cap + 1)
        .read_to_end(&mut buf)
        .context("Failed to read process output")?;
    let truncated = buf.len() as u64 > cap;
    if truncated {
        buf.truncate(cap as usize);
    }
    let text = String::from_utf8_lossy(&buf).into_owned();
    Ok((text, truncated))
}

#[cfg(test)]
mod tests {
    use std::io::Cursor;

    use super::read_capped;

    #[test]
    fn read_capped_no_truncation() {
        let data = Cursor::new(b"hello");
        let (text, truncated) = read_capped(Some(data), 100).unwrap();
        assert_eq!(text, "hello");
        assert!(!truncated);
    }

    #[test]
    fn read_capped_exact_fit() {
        let data = Cursor::new(b"abcde");
        let (text, truncated) = read_capped(Some(data), 5).unwrap();
        assert_eq!(text, "abcde");
        assert!(!truncated);
    }

    #[test]
    fn read_capped_truncates() {
        let data = Cursor::new(b"abcdefghij");
        let (text, truncated) = read_capped(Some(data), 4).unwrap();
        assert_eq!(text, "abcd");
        assert!(truncated);
    }

    #[test]
    fn read_capped_none_pipe() {
        let (text, truncated) = read_capped(None::<Cursor<&[u8]>>, 100).unwrap();
        assert_eq!(text, "");
        assert!(!truncated);
    }
}
