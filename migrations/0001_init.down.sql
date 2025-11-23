-- Migration: 0001_init.down.sql
-- Rollback initial database schema
-- Drops all tables and types created in the up migration

DROP TABLE IF EXISTS pull_request_reviewers;
DROP TABLE IF EXISTS pull_requests;
DROP TYPE IF EXISTS pull_request_status;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS teams;

