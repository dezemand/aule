-- Agent types (exploration, development, etc.)
CREATE TABLE aule.agent_types (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Insert default agent types
INSERT INTO aule.agent_types (name, description) VALUES
    ('exploration', 'Explores and understands codebases, gathers context'),
    ('research', 'Researches solutions, best practices, and approaches'),
    ('architecture', 'Designs system architecture and technical specifications'),
    ('development', 'Implements features and writes code'),
    ('documentation', 'Writes and maintains documentation'),
    ('integration', 'Integrates components and ensures they work together');

-- Tasks
CREATE TABLE aule.tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES aule.projects(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    description TEXT,
    type TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'ready',
    stage_key TEXT,
    priority INT DEFAULT 0,
    labels TEXT[],
    assignee TEXT,
    agent_type_id UUID REFERENCES aule.agent_types(id),
    
    -- Execution control (lease model for distributed execution)
    claimed_by TEXT,
    lease_until TIMESTAMPTZ,
    attempt_id TEXT,
    
    -- Context for agent execution
    context TEXT,
    system_prompt TEXT,
    allowed_tools TEXT[],
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Agent instances (running/completed agent executions)
CREATE TABLE aule.agent_instances (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES aule.projects(id) ON DELETE CASCADE,
    agent_type_id UUID NOT NULL REFERENCES aule.agent_types(id),
    task_id UUID REFERENCES aule.tasks(id) ON DELETE SET NULL,
    status TEXT NOT NULL DEFAULT 'idle',
    
    -- Execution metadata
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    result TEXT,
    error TEXT,
    
    -- Token usage tracking
    input_tokens INT DEFAULT 0,
    output_tokens INT DEFAULT 0,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Agent execution logs (for streaming updates)
CREATE TABLE aule.agent_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_instance_id UUID NOT NULL REFERENCES aule.agent_instances(id) ON DELETE CASCADE,
    log_type TEXT NOT NULL,  -- 'text', 'tool_call', 'tool_result', 'error'
    content TEXT,
    tool_name TEXT,
    tool_input JSONB,
    tool_output TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Indexes
CREATE INDEX idx_tasks_project_id ON aule.tasks(project_id);
CREATE INDEX idx_tasks_status ON aule.tasks(status);
CREATE INDEX idx_tasks_type ON aule.tasks(type);
CREATE INDEX idx_tasks_claimed_by ON aule.tasks(claimed_by) WHERE claimed_by IS NOT NULL;

CREATE INDEX idx_agent_instances_project_id ON aule.agent_instances(project_id);
CREATE INDEX idx_agent_instances_task_id ON aule.agent_instances(task_id);
CREATE INDEX idx_agent_instances_status ON aule.agent_instances(status);

CREATE INDEX idx_agent_logs_instance_id ON aule.agent_logs(agent_instance_id);
CREATE INDEX idx_agent_logs_created_at ON aule.agent_logs(created_at);

-- Triggers for updated_at
CREATE TRIGGER update_tasks_updated_at
    BEFORE UPDATE ON aule.tasks
    FOR EACH ROW EXECUTE FUNCTION aule.update_updated_at();

CREATE TRIGGER update_agent_instances_updated_at
    BEFORE UPDATE ON aule.agent_instances
    FOR EACH ROW EXECUTE FUNCTION aule.update_updated_at();
