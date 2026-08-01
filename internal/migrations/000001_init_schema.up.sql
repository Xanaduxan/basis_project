CREATE TABLE users
(
    id            BIGINT AUTO_INCREMENT PRIMARY KEY,
    name          VARCHAR(100) NOT NULL,
    email         VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
        ON UPDATE CURRENT_TIMESTAMP,

    CONSTRAINT uq_users_email UNIQUE (email)
);

CREATE TABLE teams
(
    id         BIGINT AUTO_INCREMENT PRIMARY KEY,
    name       VARCHAR(150) NOT NULL,
    created_by BIGINT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
        ON UPDATE CURRENT_TIMESTAMP,

    INDEX idx_teams_created_by (created_by),

    CONSTRAINT fk_teams_created_by
        FOREIGN KEY (created_by)
            REFERENCES users (id)
            ON DELETE RESTRICT
);

CREATE TABLE team_members
(
    team_id    BIGINT NOT NULL,
    user_id    BIGINT NOT NULL,
    role       VARCHAR(20) NOT NULL DEFAULT 'member',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
        ON UPDATE CURRENT_TIMESTAMP,

    PRIMARY KEY (team_id, user_id),

    INDEX idx_team_members_user_id (user_id),

    CONSTRAINT chk_team_members_role
        CHECK (role IN ('owner', 'admin', 'member')),

    CONSTRAINT fk_team_members_team
        FOREIGN KEY (team_id)
            REFERENCES teams (id)
            ON DELETE CASCADE,

    CONSTRAINT fk_team_members_user
        FOREIGN KEY (user_id)
            REFERENCES users (id)
            ON DELETE CASCADE
);

CREATE TABLE tasks
(
    id           BIGINT AUTO_INCREMENT PRIMARY KEY,
    team_id      BIGINT NOT NULL,
    title        VARCHAR(255) NOT NULL,
    description  TEXT,
    status       VARCHAR(20) NOT NULL DEFAULT 'todo',
    assignee_id  BIGINT,
    created_by   BIGINT NOT NULL,
    completed_at DATETIME,
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
        ON UPDATE CURRENT_TIMESTAMP,

    INDEX idx_tasks_team_filters
        (team_id, status, assignee_id, id),

    INDEX idx_tasks_assignee_id
        (assignee_id),

    INDEX idx_tasks_created_by
        (created_by),

    INDEX idx_tasks_recent_done
        (status, completed_at, team_id),

    INDEX idx_tasks_monthly_creators
        (created_at, team_id, created_by),

    CONSTRAINT chk_tasks_status
        CHECK (status IN ('todo', 'in_progress', 'done')),

    CONSTRAINT fk_tasks_assignee
        FOREIGN KEY (assignee_id)
            REFERENCES users (id)
            ON DELETE SET NULL,

    CONSTRAINT fk_tasks_team
        FOREIGN KEY (team_id)
            REFERENCES teams (id)
            ON DELETE CASCADE,

    CONSTRAINT fk_tasks_created_by
        FOREIGN KEY (created_by)
            REFERENCES users (id)
            ON DELETE RESTRICT
);

CREATE TABLE task_history
(
    id         BIGINT AUTO_INCREMENT PRIMARY KEY,
    task_id    BIGINT NOT NULL,
    changed_by BIGINT NOT NULL,
    field_name VARCHAR(64) NOT NULL,
    old_value  TEXT,
    new_value  TEXT,
    changed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    INDEX idx_task_history_task_changed_at
        (task_id, changed_at),

    INDEX idx_task_history_changed_by
        (changed_by),

    CONSTRAINT fk_task_history_task
        FOREIGN KEY (task_id)
            REFERENCES tasks (id)
            ON DELETE CASCADE,

    CONSTRAINT fk_task_history_changed_by
        FOREIGN KEY (changed_by)
            REFERENCES users (id)
            ON DELETE RESTRICT
);

CREATE TABLE task_comments
(
    id         BIGINT AUTO_INCREMENT PRIMARY KEY,
    task_id    BIGINT NOT NULL,
    user_id    BIGINT NOT NULL,
    content    TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
        ON UPDATE CURRENT_TIMESTAMP,

    INDEX idx_task_comments_task_created_at
        (task_id, created_at),

    INDEX idx_task_comments_user_id
        (user_id),

    CONSTRAINT fk_task_comments_task
        FOREIGN KEY (task_id)
            REFERENCES tasks (id)
            ON DELETE CASCADE,

    CONSTRAINT fk_task_comments_user
        FOREIGN KEY (user_id)
            REFERENCES users (id)
            ON DELETE RESTRICT
);