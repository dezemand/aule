DROP TRIGGER IF EXISTS update_projects_updated_at ON aule.projects;
DROP TRIGGER IF EXISTS update_users_updated_at ON aule.users;
DROP FUNCTION IF EXISTS aule.update_updated_at();

DROP TABLE IF EXISTS aule.project_members;
DROP TABLE IF EXISTS aule.projects;
DROP TABLE IF EXISTS aule.refresh_tokens;
DROP TABLE IF EXISTS aule.user_identities;
DROP TABLE IF EXISTS aule.users;

DROP SCHEMA IF EXISTS aule;
