-- 发音纠正 v2：会话与评测记录
CREATE TABLE pronunciation_sessions (
    id             VARCHAR(36)  NOT NULL PRIMARY KEY,
    user_id        VARCHAR(36)  NOT NULL,
    unit_id        VARCHAR(50)  NOT NULL,
    current_index  INT          NOT NULL DEFAULT 1,
    status         VARCHAR(20)  NOT NULL DEFAULT 'ongoing',
    created_at     TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at     TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP
                                  ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_user_id     (user_id),
    INDEX idx_unit_status (unit_id, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE pronunciation_records (
    id              VARCHAR(36)   NOT NULL PRIMARY KEY,
    session_id      VARCHAR(36)   NOT NULL,
    user_id         VARCHAR(36)   NOT NULL,
    unit_id         VARCHAR(50)   NOT NULL,
    item_id         INT           NOT NULL,
    content         VARCHAR(500)  NOT NULL,
    practice_type   VARCHAR(20)   NOT NULL,
    raw_score       DECIMAL(3,1)  NOT NULL,
    stars           INT           NOT NULL,
    problem_words   JSON,
    user_audio_url  VARCHAR(500),
    ai_encourage    TEXT,
    ai_problem_tip  TEXT,
    ai_suggestion   TEXT,
    ai_audio_url    VARCHAR(500),
    created_at      TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_session_item (session_id, item_id),
    INDEX idx_user_unit    (user_id, unit_id),
    INDEX idx_created_at   (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
