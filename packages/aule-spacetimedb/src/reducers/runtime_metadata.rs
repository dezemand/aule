//! Runtime metadata reducers: profile, platform info, and resource samples.

use spacetimedb::{reducer, ReducerContext, Table, TimeDuration};

use crate::tables::{
    agent_runtime, runtime_platform_info, runtime_profile, runtime_resource_sample,
    runtime_resource_sample_prune_schedule, RuntimeEnvironment, RuntimePlatformInfo,
    RuntimeProfile, RuntimeResourceSample, RuntimeResourceSamplePruneSchedule,
};

const RUNTIME_RESOURCE_SAMPLE_RETENTION_SECONDS: u64 = 24 * 60 * 60;
const RUNTIME_RESOURCE_SAMPLE_ROLLUP_INTERVAL_SECONDS: u64 = 5 * 60;
const RUNTIME_RESOURCE_SAMPLE_PRUNE_BATCH_SIZE: usize = 500;

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
    if runtime_instance_id.trim().is_empty() {
        return Err("runtime_instance_id must not be empty".to_string());
    }
    if runtime_name.trim().is_empty() {
        return Err("runtime_name must not be empty".to_string());
    }
    if runtime_version.trim().is_empty() {
        return Err("runtime_version must not be empty".to_string());
    }
    if os.trim().is_empty() {
        return Err("os must not be empty".to_string());
    }
    if arch.trim().is_empty() {
        return Err("arch must not be empty".to_string());
    }

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

    prune_runtime_resource_samples_internal(
        ctx,
        RUNTIME_RESOURCE_SAMPLE_RETENTION_SECONDS,
        RUNTIME_RESOURCE_SAMPLE_PRUNE_BATCH_SIZE,
    );

    Ok(())
}

#[reducer]
pub fn prune_runtime_resource_samples(
    ctx: &ReducerContext,
    schedule: RuntimeResourceSamplePruneSchedule,
) -> Result<(), String> {
    prune_runtime_resource_samples_internal(
        ctx,
        schedule.retention_seconds,
        schedule.prune_batch_size as usize,
    );
    schedule_next_runtime_resource_sample_prune(
        ctx,
        schedule.retention_seconds,
        schedule.prune_batch_size,
    );
    Ok(())
}

pub(crate) fn schedule_next_runtime_resource_sample_prune(
    ctx: &ReducerContext,
    retention_seconds: u64,
    prune_batch_size: u32,
) {
    let interval = TimeDuration::from_micros(
        i64::try_from(RUNTIME_RESOURCE_SAMPLE_ROLLUP_INTERVAL_SECONDS)
            .unwrap_or(0)
            .saturating_mul(1_000_000),
    );

    ctx.db
        .runtime_resource_sample_prune_schedule()
        .insert(RuntimeResourceSamplePruneSchedule {
            scheduled_id: 0,
            scheduled_at: interval.into(),
            retention_seconds,
            prune_batch_size,
        });
}

fn prune_runtime_resource_samples_internal(
    ctx: &ReducerContext,
    retention_seconds: u64,
    prune_batch_size: usize,
) {
    let retention_micros = i128::from(retention_seconds).saturating_mul(1_000_000);
    let cutoff_micros =
        i128::from(ctx.timestamp.to_micros_since_unix_epoch()).saturating_sub(retention_micros);

    let stale_ids: Vec<u64> = ctx
        .db
        .runtime_resource_sample()
        .iter()
        .filter_map(|sample| {
            let sampled_at = i128::from(sample.sampled_at.to_micros_since_unix_epoch());
            if sampled_at < cutoff_micros {
                Some(sample.id)
            } else {
                None
            }
        })
        .take(prune_batch_size)
        .collect();

    for id in stale_ids {
        ctx.db.runtime_resource_sample().id().delete(id);
    }
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
