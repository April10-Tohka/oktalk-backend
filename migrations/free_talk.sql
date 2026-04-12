-- ============================================================
-- Free Talk 会话相关表
-- ============================================================

-- 表1：free_talk_sessions
-- 每次 WebSocket 连接对应一行，记录会话基本信息
CREATE TABLE IF NOT EXISTS free_talk_sessions (
    id            BIGSERIAL    PRIMARY KEY,
    session_id    VARCHAR(64)  NOT NULL UNIQUE,  -- 与 service 层 conversationID 保持一致
    user_id       VARCHAR(64)  NOT NULL,
    started_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    ended_at      TIMESTAMPTZ,                   -- Session 正常/异常结束后更新
    turn_count    INTEGER      NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_fts_user_id    ON free_talk_sessions (user_id);
CREATE INDEX IF NOT EXISTS idx_fts_started_at ON free_talk_sessions (started_at DESC);

COMMENT ON TABLE  free_talk_sessions            IS 'Free Talk 实时对话会话表，一行对应一次 WebSocket 连接';
COMMENT ON COLUMN free_talk_sessions.session_id IS '会话唯一标识，由 Handler 层在建立 WebSocket 时生成并传入 Session';
COMMENT ON COLUMN free_talk_sessions.ended_at   IS 'Session.Run() 返回后由 Handler 层写入；NULL 表示异常断开未更新';
COMMENT ON COLUMN free_talk_sessions.turn_count IS 'AI 回复条数（含开场白、静默触发的主动发言），Session 结束后更新';

-- ============================================================

-- 表2：free_talk_messages
-- 某次会话下的每一条消息，session 期间实时追加写入
-- role: 'user'=孩子(ASR识别文本)  'assistant'=AI Mia(LLM完整回复)
CREATE TABLE IF NOT EXISTS free_talk_messages (
    id            BIGSERIAL    PRIMARY KEY,
    session_id    VARCHAR(64)  NOT NULL REFERENCES free_talk_sessions (session_id) ON DELETE CASCADE,
    seq           INTEGER      NOT NULL,          -- 消息在本次会话内的顺序号，从 1 开始
    role          VARCHAR(16)  NOT NULL CHECK (role IN ('user', 'assistant')),
    content       TEXT         NOT NULL,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ftm_session_seq ON free_talk_messages (session_id, seq);

COMMENT ON TABLE  free_talk_messages         IS 'Free Talk 会话消息明细表，session 期间实时写入';
COMMENT ON COLUMN free_talk_messages.seq     IS '消息顺序号，前端按 seq 升序渲染对话气泡';
COMMENT ON COLUMN free_talk_messages.content IS '用户侧为 ASR 识别文本；AI 侧为 LLM 完整回复（非流式 token 拼接结果）';