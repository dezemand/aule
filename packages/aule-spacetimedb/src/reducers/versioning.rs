//! Agent type and version management reducers.

use spacetimedb::{reducer, ReducerContext, Table};

use crate::tables::{agent_type, agent_type_version, AgentType, AgentTypeVersion, VersionStatus};

/// Create a new agent type (e.g. "builder", "researcher").
#[reducer]
pub fn create_agent_type(
    ctx: &ReducerContext,
    name: String,
    description: String,
) -> Result<(), String> {
    let name = name.trim().to_string();
    if name.is_empty() {
        return Err("Agent type name must not be empty".to_string());
    }

    // Check for duplicate name
    for existing in ctx.db.agent_type().iter() {
        if existing.name == name {
            return Err(format!("Agent type '{name}' already exists"));
        }
    }

    ctx.db.agent_type().insert(AgentType {
        id: 0,
        name: name.clone(),
        description,
        created_by: ctx.sender(),
        created_at: ctx.timestamp,
    });

    log::info!("Agent type created: {name}");
    Ok(())
}

/// Create a new version of an agent type with a system prompt.
/// New versions start in Draft status.
#[reducer]
pub fn create_agent_type_version(
    ctx: &ReducerContext,
    agent_type_id: u64,
    version: String,
    system_prompt: String,
) -> Result<(), String> {
    // Verify agent type exists
    if ctx.db.agent_type().id().find(agent_type_id).is_none() {
        return Err(format!("Agent type {agent_type_id} not found"));
    }

    let version = version.trim().to_string();
    if version.is_empty() {
        return Err("Version string must not be empty".to_string());
    }
    if system_prompt.trim().is_empty() {
        return Err("System prompt must not be empty".to_string());
    }

    // Check for duplicate version string within this agent type
    for existing in ctx.db.agent_type_version().iter() {
        if existing.agent_type_id == agent_type_id && existing.version == version {
            return Err(format!(
                "Version '{version}' already exists for agent type {agent_type_id}"
            ));
        }
    }

    ctx.db.agent_type_version().insert(AgentTypeVersion {
        id: 0,
        agent_type_id,
        version: version.clone(),
        system_prompt,
        status: VersionStatus::Draft,
        created_by: ctx.sender(),
        created_at: ctx.timestamp,
    });

    log::info!("Agent type version created: type={agent_type_id}, version={version}");
    Ok(())
}

/// Activate an agent type version, making it available for use.
/// Only Draft or Testing versions can be activated.
#[reducer]
pub fn activate_agent_type_version(ctx: &ReducerContext, version_id: u64) -> Result<(), String> {
    let version = ctx
        .db
        .agent_type_version()
        .id()
        .find(version_id)
        .ok_or(format!("Version {version_id} not found"))?;

    match version.status {
        VersionStatus::Draft | VersionStatus::Testing => {}
        _ => {
            return Err(format!(
                "Cannot activate version in {:?} status",
                version.status
            ));
        }
    }

    ctx.db.agent_type_version().id().update(AgentTypeVersion {
        status: VersionStatus::Active,
        ..version
    });

    log::info!("Agent type version activated: {version_id}");
    Ok(())
}
