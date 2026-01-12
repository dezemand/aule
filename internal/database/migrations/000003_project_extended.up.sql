-- Migration: Extend projects table with full project information model
-- See docs/objects/project.md for specification

-- Section 1: Add status field (active | paused | archived)
ALTER TABLE aule.projects ADD COLUMN status TEXT NOT NULL DEFAULT 'active';

-- Section 2: Purpose & Intent (JSONB)
-- Contains: goal, problem_statement, non_goals[], expected_value, time_horizon
ALTER TABLE aule.projects ADD COLUMN purpose JSONB;

-- Migrate existing goal data to purpose.goal
UPDATE aule.projects 
SET purpose = jsonb_build_object('goal', goal)
WHERE goal IS NOT NULL AND goal != '';

-- Drop old goal column after migration
ALTER TABLE aule.projects DROP COLUMN goal;

-- Section 3: Scope & Boundaries (JSONB)
-- Contains: in_scope[], out_of_scope[], assumptions[], constraints[]
ALTER TABLE aule.projects ADD COLUMN scope JSONB;

-- Section 5: Governance & Autonomy (JSONB)
-- Contains: autonomy_level, human_in_the_loop[], review_strictness, decision_authority, escalation_rules[]
ALTER TABLE aule.projects ADD COLUMN governance JSONB;

-- Section 6: Task Model Configuration (JSONB)
-- Contains: allowed_task_types[], custom_stage_overrides{}, default_priorities{}, wip_limits{}
ALTER TABLE aule.projects ADD COLUMN task_config JSONB;

-- Section 7: Agent Configuration (JSONB)
-- Contains: allowed_agent_types[], trust_level, runtime_permissions[], max_parallel_agents, budget_limits{}
ALTER TABLE aule.projects ADD COLUMN agent_config JSONB;

-- Section 4: Extend project_members with permissions
ALTER TABLE aule.project_members ADD COLUMN permissions JSONB;

-- Add index on project status for filtering
CREATE INDEX idx_projects_status ON aule.projects(status);

-- Section 8: Artefact Attachments - Git Repositories
CREATE TABLE aule.project_repositories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES aule.projects(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    purpose TEXT NOT NULL,  -- code | docs | infra | examples
    default_branch TEXT NOT NULL DEFAULT 'main',
    allowed_paths JSONB,    -- { read: [], write: [] }
    branch_naming_convention TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (project_id, url)
);

CREATE INDEX idx_project_repositories_project_id ON aule.project_repositories(project_id);

-- Apply updated_at trigger to project_repositories
CREATE TRIGGER update_project_repositories_updated_at
    BEFORE UPDATE ON aule.project_repositories
    FOR EACH ROW
    EXECUTE FUNCTION aule.update_updated_at();

-- Section 8: Artefact Attachments - Wiki Spaces
CREATE TABLE aule.project_wiki_spaces (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES aule.projects(id) ON DELETE CASCADE,
    space_id TEXT NOT NULL,
    access_mode TEXT NOT NULL DEFAULT 'read',  -- read | write
    page_prefixes TEXT[],
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (project_id, space_id)
);

CREATE INDEX idx_project_wiki_spaces_project_id ON aule.project_wiki_spaces(project_id);

-- Apply updated_at trigger to project_wiki_spaces
CREATE TRIGGER update_project_wiki_spaces_updated_at
    BEFORE UPDATE ON aule.project_wiki_spaces
    FOR EACH ROW
    EXECUTE FUNCTION aule.update_updated_at();
