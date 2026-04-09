-- PostgreSQL init script: create schemas for each microservice
-- This runs automatically when the postgres container starts for the first time

CREATE SCHEMA IF NOT EXISTS auth_service;
CREATE SCHEMA IF NOT EXISTS attendance_service;
CREATE SCHEMA IF NOT EXISTS leave_service;
CREATE SCHEMA IF NOT EXISTS analytics_service;
CREATE SCHEMA IF NOT EXISTS org_service;

-- Grant usage to the app user
GRANT ALL ON SCHEMA auth_service TO app;
GRANT ALL ON SCHEMA attendance_service TO app;
GRANT ALL ON SCHEMA leave_service TO app;
GRANT ALL ON SCHEMA analytics_service TO app;
GRANT ALL ON SCHEMA org_service TO app;
