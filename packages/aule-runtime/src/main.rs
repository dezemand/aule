mod config;
mod llm_client;
mod runtime;
mod shell;

use anyhow::Result;
use config::RuntimeConfig;

fn main() -> Result<()> {
    env_logger::init();

    let config = RuntimeConfig::from_env()?;
    runtime::run(config)
}
