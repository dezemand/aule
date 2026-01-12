-- Rollback: Remove extended project fields

-- Drop wiki spaces table
DROP TRIGGER IF EXISTS update_project_wiki_spaces_updated_at ON aule.project_wiki_spaces;
DROP TABLE IF EXISTS aule.project_wiki_spaces;

-- Drop repositories table
DROP TRIGGER IF EXISTS update_project_repositories_updated_at ON aule.project_repositories;
DROP TABLE IF EXISTS aule.project_repositories;

-- Remove permissions from project_members
ALTER TABLE aule.project_members DROP COLUMN IF EXISTS permissions;

-- Remove index on status
DROP INDEX IF EXISTS aule.idx_projects_status;

-- Remove JSONB columns from projects
ALTER TABLE aule.projects DROP COLUMN IF EXISTS agent_config;
ALTER TABLE aule.projects DROP COLUMN IF EXISTS task_config;
ALTER TABLE aule.projects DROP COLUMN IF EXISTS governance;
ALTER TABLE aule.projects DROP COLUMN IF EXISTS scope;

-- Restore goal column and migrate data back from purpose
ALTER TABLE aule.projects ADD COLUMN goal TEXT;

UPDATE aule.projects 
SET goal = purpose->>'goal'
WHERE purpose IS NOT NULL AND purpose->>'goal' IS NOT NULL;

ALTER TABLE aule.projects DROP COLUMN IF EXISTS purpose;

-- Remove status column
ALTER TABLE aule.projects DROP COLUMN IF EXISTS status;
