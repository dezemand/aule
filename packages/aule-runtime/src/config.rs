use std::{env, fmt, time::Duration};

use anyhow::{Context, Result, bail};

#[derive(Clone)]
pub struct RuntimeConfig {
    pub spacetimedb_uri: String,
    pub spacetimedb_db_name: String,
    pub runtime_name: String,
    pub agent_version: String,
    pub openai_api_key: String,
    pub openai_model: String,
    pub heartbeat_interval: Duration,
    pub resource_sample_interval: Duration,
    pub shell_timeout: Duration,
    pub shell_output_limit_bytes: usize,
    pub max_steps_per_task: u32,
}

impl fmt::Debug for RuntimeConfig {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.debug_struct("RuntimeConfig")
            .field("spacetimedb_uri", &self.spacetimedb_uri)
            .field("spacetimedb_db_name", &self.spacetimedb_db_name)
            .field("runtime_name", &self.runtime_name)
            .field("agent_version", &self.agent_version)
            .field("openai_api_key", &"[REDACTED]")
            .field("openai_model", &self.openai_model)
            .field("heartbeat_interval", &self.heartbeat_interval)
            .field("resource_sample_interval", &self.resource_sample_interval)
            .field("shell_timeout", &self.shell_timeout)
            .field("shell_output_limit_bytes", &self.shell_output_limit_bytes)
            .field("max_steps_per_task", &self.max_steps_per_task)
            .finish()
    }
}

impl RuntimeConfig {
    pub fn from_env() -> Result<Self> {
        let spacetimedb_uri =
            env::var("SPACETIMEDB_URI").unwrap_or_else(|_| "http://localhost:3000".to_string());
        let spacetimedb_db_name =
            env::var("SPACETIMEDB_DB_NAME").unwrap_or_else(|_| "aule".to_string());
        let runtime_name =
            env::var("AULE_RUNTIME_NAME").unwrap_or_else(|_| "runtime-01".to_string());

        let agent_version = required("AULE_AGENT_VERSION")?;
        let openai_api_key = required("OPENAI_API_KEY")?;
        let openai_model = env::var("OPENAI_MODEL").unwrap_or_else(|_| "gpt-4.1-mini".to_string());

        let heartbeat_interval =
            Duration::from_secs(parse_u64_with_default("AULE_HEARTBEAT_SECONDS", 10)?);
        let resource_sample_interval =
            Duration::from_secs(parse_u64_with_default("AULE_RESOURCE_SAMPLE_SECONDS", 30)?);
        let shell_timeout =
            Duration::from_secs(parse_u64_with_default("AULE_SHELL_TIMEOUT_SECONDS", 30)?);

        let shell_output_limit_bytes =
            parse_usize_with_default("AULE_SHELL_OUTPUT_LIMIT_BYTES", 50_000)?;
        let max_steps_per_task = parse_u32_with_default("AULE_MAX_STEPS_PER_TASK", 24)?;

        if runtime_name.trim().is_empty() {
            bail!("AULE_RUNTIME_NAME must not be empty");
        }

        Ok(Self {
            spacetimedb_uri,
            spacetimedb_db_name,
            runtime_name,
            agent_version,
            openai_api_key,
            openai_model,
            heartbeat_interval,
            resource_sample_interval,
            shell_timeout,
            shell_output_limit_bytes,
            max_steps_per_task,
        })
    }
}

fn required(key: &str) -> Result<String> {
    let value = env::var(key).with_context(|| format!("Missing required env var {key}"))?;
    let trimmed = value.trim();
    if trimmed.is_empty() {
        bail!("{key} must not be empty");
    }
    Ok(trimmed.to_string())
}

fn parse_u64_with_default(key: &str, default: u64) -> Result<u64> {
    match env::var(key) {
        Ok(value) => value
            .trim()
            .parse::<u64>()
            .with_context(|| format!("{key} must be a valid u64")),
        Err(_) => Ok(default),
    }
}

fn parse_u32_with_default(key: &str, default: u32) -> Result<u32> {
    match env::var(key) {
        Ok(value) => value
            .trim()
            .parse::<u32>()
            .with_context(|| format!("{key} must be a valid u32")),
        Err(_) => Ok(default),
    }
}

fn parse_usize_with_default(key: &str, default: usize) -> Result<usize> {
    match env::var(key) {
        Ok(value) => value
            .trim()
            .parse::<usize>()
            .with_context(|| format!("{key} must be a valid usize")),
        Err(_) => Ok(default),
    }
}
