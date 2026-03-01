//! Runtime metadata reducers: profile, platform info, and resource samples.

use spacetimedb::{ReducerContext, Table, reducer};

use crate::tables::{
    RuntimeEnvironment, RuntimePlatformInfo, RuntimeProfile, RuntimeResourceSample, agent_runtime,
    runtime_platform_info, runtime_profile, runtime_resource_sample,
};

#[reducer]
pub fn upsert_runtime_profile(
    ctx: &ReducerContext,
    runtime_instance_id: String,
    runtime_name: String,
    runtime_version: String,
    git_sha: Option<String>,
    hostname: Option<String>,
    os: String,
    arch: String,
) -> Result<(), String> {
    let sender = ctx.sender();
    ensure_registered_runtime(ctx, sender)?;

    if let Some(existing) = ctx.db.runtime_profile().runtime_identity().find(sender) {
        ctx.db
            .runtime_profile()
            .runtime_identity()
            .update(RuntimeProfile {
                runtime_identity: sender,
                runtime_instance_id,
                runtime_name,
                runtime_version,
                git_sha,
                hostname,
                os,
                arch,
                started_at: existing.started_at,
                updated_at: ctx.timestamp,
            });
    } else {
        ctx.db.runtime_profile().insert(RuntimeProfile {
            runtime_identity: sender,
            runtime_instance_id,
            runtime_name,
            runtime_version,
            git_sha,
            hostname,
            os,
            arch,
            started_at: ctx.timestamp,
            updated_at: ctx.timestamp,
        });
    }

    Ok(())
}

#[reducer]
pub fn upsert_runtime_platform_info(
    ctx: &ReducerContext,
    environment: RuntimeEnvironment,
    process_id: Option<u32>,
    container_id: Option<String>,
    image: Option<String>,
    image_digest: Option<String>,
    cluster: Option<String>,
    namespace: Option<String>,
    pod_name: Option<String>,
    pod_uid: Option<String>,
    node_name: Option<String>,
    workload_kind: Option<String>,
    workload_name: Option<String>,
    container_name: Option<String>,
    restart_count: Option<u32>,
) -> Result<(), String> {
    let sender = ctx.sender();
    ensure_registered_runtime(ctx, sender)?;

    let row = RuntimePlatformInfo {
        runtime_identity: sender,
        environment,
        process_id,
        container_id,
        image,
        image_digest,
        cluster,
        namespace,
        pod_name,
        pod_uid,
        node_name,
        workload_kind,
        workload_name,
        container_name,
        restart_count,
        updated_at: ctx.timestamp,
    };

    if ctx
        .db
        .runtime_platform_info()
        .runtime_identity()
        .find(sender)
        .is_some()
    {
        ctx.db
            .runtime_platform_info()
            .runtime_identity()
            .update(row);
    } else {
        ctx.db.runtime_platform_info().insert(row);
    }

    Ok(())
}

#[reducer]
pub fn insert_runtime_resource_sample(
    ctx: &ReducerContext,
    cpu_millicores: Option<u32>,
    memory_rss_bytes: u64,
    memory_working_set_bytes: Option<u64>,
    threads: Option<u32>,
    open_fds: Option<u32>,
) -> Result<(), String> {
    let sender = ctx.sender();
    ensure_registered_runtime(ctx, sender)?;

    ctx.db
        .runtime_resource_sample()
        .insert(RuntimeResourceSample {
            id: 0,
            runtime_identity: sender,
            sampled_at: ctx.timestamp,
            cpu_millicores,
            memory_rss_bytes,
            memory_working_set_bytes,
            threads,
            open_fds,
        });

    Ok(())
}

fn ensure_registered_runtime(
    ctx: &ReducerContext,
    sender: spacetimedb::Identity,
) -> Result<(), String> {
    if ctx.db.agent_runtime().identity().find(sender).is_none() {
        return Err("Runtime not registered".to_string());
    }
    Ok(())
}
