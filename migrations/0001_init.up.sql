-- Migration: 0001_init.up.sql
-- Initial database schema creation
-- Creates all tables, types, and constraints for the code review service

CREATE TABLE teams (
    team_name VARCHAR(100) PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE users (
    user_id VARCHAR(100) PRIMARY KEY,
    username VARCHAR(100) NOT NULL,
    team_name VARCHAR(100) NOT NULL REFERENCES teams(team_name) ON DELETE CASCADE,
    is_active BOOLEAN NOT NULL DEFAULT true
);

CREATE TYPE pull_request_status AS ENUM ('OPEN', 'MERGED');

CREATE TABLE pull_requests (
    pull_request_id VARCHAR(100) PRIMARY KEY,
    pull_request_name VARCHAR(255) NOT NULL,
    author_id VARCHAR(100) NOT NULL REFERENCES users(user_id),
    status pull_request_status NOT NULL DEFAULT 'OPEN',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    merged_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE pull_request_reviewers (
    pull_request_id VARCHAR(100) REFERENCES pull_requests(pull_request_id) ON DELETE CASCADE,
    user_id VARCHAR(100) REFERENCES users(user_id) ON DELETE CASCADE,
    assigned_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (pull_request_id, user_id)
);
