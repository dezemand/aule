DROP TRIGGER IF EXISTS update_agent_instances_updated_at ON aule.agent_instances;
DROP TRIGGER IF EXISTS update_tasks_updated_at ON aule.tasks;

DROP TABLE IF EXISTS aule.agent_logs;
DROP TABLE IF EXISTS aule.agent_instances;
DROP TABLE IF EXISTS aule.tasks;
DROP TABLE IF EXISTS aule.agent_types;
