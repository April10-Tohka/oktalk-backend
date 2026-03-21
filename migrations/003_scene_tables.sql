-- 场景引导：会话与消息表
CREATE TABLE scene_sessions (
    id            VARCHAR(36)  NOT NULL PRIMARY KEY,
    user_id       VARCHAR(36)  NOT NULL,
    scene_id      VARCHAR(50)  NOT NULL,
    current_step  INT          NOT NULL DEFAULT 1,
    status        VARCHAR(20)  NOT NULL DEFAULT 'active',
    created_at    TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP
                               ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_user_id (user_id),
    INDEX idx_status  (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE scene_messages (
    id              VARCHAR(36)   NOT NULL PRIMARY KEY,
    session_id      VARCHAR(36)   NOT NULL,
    user_id         VARCHAR(36)   NOT NULL,
    scene_id        VARCHAR(50)   NOT NULL,
    step_id         INT           NOT NULL,
    attempt         INT           NOT NULL DEFAULT 1,
    user_text       TEXT,
    user_audio_url  VARCHAR(500),
    match_result    VARCHAR(20)   NOT NULL,
    ai_reply_text   TEXT,
    ai_audio_url    VARCHAR(500),
    llm_status      VARCHAR(10),
    step_advanced   BOOLEAN       NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_session_id (session_id),
    INDEX idx_user_step  (user_id, scene_id, step_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
