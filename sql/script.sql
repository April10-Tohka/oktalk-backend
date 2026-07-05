create table free_talk_messages
(
    id         varchar(36) not null
        primary key,
    session_id varchar(36) not null,
    seq        bigint      not null,
    role       varchar(20) not null,
    content    text        null,
    created_at datetime(3) null
);

create index idx_free_talk_messages_session_id
    on free_talk_messages (session_id);

create table free_talk_sessions
(
    id         varchar(36)      not null
        primary key,
    session_id varchar(36)      not null,
    user_id    varchar(36)      not null,
    started_at datetime         null,
    ended_at   datetime         null,
    turn_count bigint default 0 null,
    created_at datetime(3)      null,
    constraint idx_free_talk_sessions_session_id
        unique (session_id)
);

create index idx_free_talk_sessions_user_id
    on free_talk_sessions (user_id);

create table pronunciation_records
(
    id              varchar(36)                not null
        primary key,
    session_id      varchar(36)                not null,
    user_id         varchar(36)                not null,
    unit_id         varchar(50)                not null,
    item_id         bigint                     not null,
    content         varchar(500)               not null,
    practice_type   varchar(20)                not null,
    raw_score       decimal(3, 1)              not null,
    stars           bigint                     not null,
    problem_words   json                       null,
    user_audio_url  varchar(500)               null,
    ai_encourage    text                       null,
    ai_problem_tip  text                       null,
    ai_suggestion   text                       null,
    ai_audio_url    varchar(500)               null,
    created_at      datetime(3)                null,
    fluency         decimal(3, 1) default 0.0  not null,
    integrity       decimal(3, 1) default 0.0  not null,
    phone_score     decimal(3, 1) default 0.0  null,
    accuracy_score  decimal(3, 1) default 0.0  not null,
    fluency_score   decimal(4, 2) default 0.00 null comment '''流利度评分''',
    integrity_score decimal(4, 2) default 0.00 null comment '''完整度评分''',
    standard_score  decimal(3, 1) default 0.0  not null,
    except_info     bigint        default 0    null comment '''异常信息''',
    is_rejected     tinyint(1)    default 0    not null,
    raw_xml         longtext                   null,
    ai_how_to_fix   text                       null,
    how_to_fix_url  varchar(500)               null,
    a_iproblem_tip  text                       null
);

create index idx_pronunciation_records_is_rejected
    on pronunciation_records (is_rejected);

create index idx_pronunciation_records_session_id
    on pronunciation_records (session_id);

create table pronunciation_sessions
(
    id            varchar(36)                   not null
        primary key,
    user_id       varchar(36)                   not null,
    unit_id       varchar(50)                   not null,
    current_index bigint      default 1         not null,
    status        varchar(20) default 'ongoing' not null,
    created_at    datetime(3)                   null,
    updated_at    datetime(3)                   null
);

create index idx_pronunciation_sessions_status
    on pronunciation_sessions (status);

create index idx_pronunciation_sessions_user_id
    on pronunciation_sessions (user_id);

create table scene_messages
(
    id             varchar(36)          not null
        primary key,
    session_id     varchar(36)          not null,
    user_id        varchar(36)          not null,
    scene_id       varchar(50)          not null,
    step_id        bigint               not null,
    attempt        bigint     default 1 not null,
    user_text      text                 null,
    user_audio_url varchar(500)         null,
    match_result   varchar(20)          not null,
    ai_reply_text  text                 null,
    ai_audio_url   varchar(500)         null,
    llm_status     varchar(10)          null,
    step_advanced  tinyint(1) default 0 null,
    created_at     datetime(3)          null
);

create index idx_scene_messages_session_id
    on scene_messages (session_id);

create table scene_sessions
(
    id           varchar(36)                  not null
        primary key,
    user_id      varchar(36)                  not null,
    scene_id     varchar(50)                  not null,
    current_step bigint      default 1        not null,
    status       varchar(20) default 'active' not null,
    created_at   datetime(3)                  null,
    updated_at   datetime(3)                  null
);

create index idx_scene_sessions_status
    on scene_sessions (status);

create index idx_scene_sessions_user_id
    on scene_sessions (user_id);

create table sessions
(
    id         varchar(36)                         not null comment '会话ID'
        primary key,
    user_id    varchar(36)                         not null comment '用户ID',
    token      varchar(255)                        not null comment '会话令牌',
    user_agent varchar(255)                        null comment 'User-Agent',
    ip         varchar(50)                         null comment '客户端IP',
    expires_at timestamp                           not null comment '过期时间',
    created_at timestamp default CURRENT_TIMESTAMP not null comment '创建时间',
    constraint uk_sessions_token
        unique (token)
)
    comment '用户登录会话' collate = utf8mb4_unicode_ci;

create index idx_sessions_expires_at
    on sessions (expires_at);

create index idx_sessions_user_id
    on sessions (user_id);

create table system_settings
(
    id           varchar(36)                                        not null
        primary key,
    config_key   varchar(100)                                       not null,
    config_value longtext                                           not null,
    config_type  enum ('string', 'int', 'float', 'json', 'boolean') not null,
    description  varchar(500)                                       null,
    is_editable  tinyint(1) default 1                               not null,
    created_at   timestamp                                          null,
    updated_at   timestamp                                          null,
    constraint idx_system_settings_config_key
        unique (config_key)
);

create table users
(
    id              varchar(36)                  not null
        primary key,
    username        varchar(100)                 not null,
    phone           varchar(20)                  null,
    avatar_url      varchar(500)                 null,
    grade           bigint                       null,
    created_at      timestamp                    null,
    updated_at      timestamp                    null,
    deleted_at      timestamp                    null,
    register_source varchar(20)                  not null,
    status          varchar(20) default 'active' not null,
    constraint idx_users_phone
        unique (phone),
    constraint idx_users_username
        unique (username)
);

create table learning_reports
(
    id                         varchar(36)                                           not null
        primary key,
    user_id                    varchar(36)                                           not null,
    report_type                enum ('weekly', 'monthly', 'custom') default 'weekly' not null,
    period_start_date          date                                                  not null,
    period_end_date            date                                                  not null,
    total_conversations        bigint                               default 0        not null,
    total_evaluations          bigint                               default 0        not null,
    total_study_minutes        bigint                               default 0        not null,
    average_conversation_score double                               default 0        not null,
    average_evaluation_score   double                               default 0        not null,
    average_accuracy_score     double                               default 0        not null,
    average_fluency_score      double                               default 0        not null,
    average_integrity_score    double                               default 0        not null,
    s_level_count              bigint                               default 0        not null,
    a_level_count              bigint                               default 0        not null,
    b_level_count              bigint                               default 0        not null,
    c_level_count              bigint                               default 0        not null,
    improvement_rate           double                               default 0        not null,
    most_practiced_topics      json                                                  null,
    problem_words              json                                                  null,
    strengths                  json                                                  null,
    weaknesses                 json                                                  null,
    recommendations            longtext                                              null,
    ai_model                   varchar(100)                                          null,
    created_at                 timestamp                                             null,
    updated_at                 timestamp                                             null,
    task_id                    varchar(100)                         default ''       null,
    content                    longtext                                              null,
    is_latest                  tinyint(1)                           default 1        not null,
    constraint fk_users_reports
        foreign key (user_id) references users (id)
);

create index idx_learning_reports_created_at
    on learning_reports (created_at);

create index idx_learning_reports_is_latest
    on learning_reports (is_latest);

create index idx_learning_reports_period_start_date
    on learning_reports (period_start_date);

create index idx_learning_reports_report_type
    on learning_reports (report_type);

create index idx_learning_reports_user_id
    on learning_reports (user_id);

create index idx_lr_user_latest
    on learning_reports (user_id, is_latest);

create table pronunciation_evaluations
(
    id                      varchar(36)                                                              not null
        primary key,
    user_id                 varchar(36)                                                              not null,
    target_text             varchar(500)                                                             not null,
    recognized_text         varchar(500)                                                             null,
    audio_url               varchar(500)                                                             null,
    audio_duration          bigint                                                                   null,
    overall_score           bigint                                                default 0          not null,
    accuracy_score          bigint                                                default 0          not null,
    fluency_score           bigint                                                default 0          not null,
    integrity_score         bigint                                                default 0          not null,
    feedback_level          enum ('S', 'A', 'B', 'C')                             default 'C'        not null,
    feedback_text           text                                                                     null,
    feedback_audio_url      varchar(500)                                                             null,
    problem_words           json                                                                     null,
    problem_word_audio_urls json                                                                     null,
    demo_sentence_audio_url varchar(500)                                                             null,
    difficulty_level        enum ('beginner', 'intermediate', 'advanced')         default 'beginner' not null,
    assessment_s_id         varchar(100)                                                             null,
    speech_assessment_json  longtext                                                                 null,
    status                  enum ('pending', 'processing', 'completed', 'failed') default 'pending'  not null,
    error_message           varchar(500)                                                             null,
    created_at              timestamp                                                                null,
    updated_at              timestamp                                                                null,
    constraint fk_users_evaluations
        foreign key (user_id) references users (id)
);

create index idx_pronunciation_evaluations_created_at
    on pronunciation_evaluations (created_at);

create index idx_pronunciation_evaluations_feedback_level
    on pronunciation_evaluations (feedback_level);

create index idx_pronunciation_evaluations_status
    on pronunciation_evaluations (status);

create index idx_pronunciation_evaluations_user_id
    on pronunciation_evaluations (user_id);

create table user_login_logs
(
    id          varchar(36)  not null
        primary key,
    user_id     varchar(36)  not null,
    login_type  varchar(20)  not null,
    result      varchar(20)  not null,
    fail_reason varchar(200) null,
    ip          varchar(45)  not null,
    platform    varchar(20)  null,
    device_id   varchar(100) null,
    user_agent  varchar(500) null,
    created_at  timestamp    null,
    constraint fk_users_login_logs
        foreign key (user_id) references users (id)
);

create index idx_user_login_logs_created_at
    on user_login_logs (created_at);

create index idx_user_login_logs_user_id
    on user_login_logs (user_id);

create table user_profiles
(
    id                       varchar(36)          not null
        primary key,
    user_id                  varchar(36)          not null,
    age                      bigint               null,
    gender                   enum ('boy', 'girl') null,
    bio                      text                 null,
    total_conversations      bigint default 0     not null,
    total_evaluations        bigint default 0     not null,
    total_reports            bigint default 0     not null,
    total_study_minutes      bigint default 0     not null,
    average_evaluation_score double default 0     not null,
    last_conversation_at     timestamp            null,
    last_evaluation_at       timestamp            null,
    created_at               timestamp            null,
    updated_at               timestamp            null,
    constraint idx_user_profiles_user_id
        unique (user_id),
    constraint fk_users_profile
        foreign key (user_id) references users (id)
);

create index idx_user_profiles_last_conversation_at
    on user_profiles (last_conversation_at);

create index idx_user_profiles_last_evaluation_at
    on user_profiles (last_evaluation_at);

create table user_wechat_bindings
(
    id                varchar(36)  not null
        primary key,
    user_id           varchar(36)  not null,
    open_id           varchar(100) not null,
    wechat_nickname   varchar(100) null,
    wechat_avatar_url varchar(500) null,
    created_at        timestamp    null,
    updated_at        timestamp    null,
    constraint idx_user_wechat_bindings_open_id
        unique (open_id),
    constraint idx_user_wechat_bindings_user_id
        unique (user_id),
    constraint fk_users_wechat_binding
        foreign key (user_id) references users (id)
);

create index idx_users_created_at
    on users (created_at);

create index idx_users_deleted_at
    on users (deleted_at);

create table voice_conversations
(
    id                varchar(36)                                                       not null
        primary key,
    user_id           varchar(36)                                                       not null,
    topic             varchar(200)                                                      not null,
    difficulty_level  enum ('beginner', 'intermediate', 'advanced') default 'beginner'  not null,
    conversation_type enum ('free_talk', 'question_answer')         default 'free_talk' not null,
    message_count     bigint                                        default 0           not null,
    duration_seconds  bigint                                        default 0           not null,
    status            enum ('active', 'completed', 'paused')        default 'active'    not null,
    summary           text                                                              null,
    score             bigint                                                            null,
    feedback          text                                                              null,
    created_at        timestamp                                                         null,
    updated_at        timestamp                                                         null,
    deleted_at        timestamp                                                         null,
    constraint fk_users_voice_conversations
        foreign key (user_id) references users (id)
);

create table conversation_messages
(
    id              varchar(36)         not null
        primary key,
    conversation_id varchar(36)         not null,
    sender_type     enum ('user', 'ai') not null,
    message_text    longtext            not null,
    audio_url       varchar(500)        null,
    audio_duration  bigint              null,
    sequence_number bigint              not null,
    latency_ms      bigint              null,
    created_at      timestamp           null,
    updated_at      timestamp           null,
    turn_id         bigint              not null,
    constraint fk_voice_conversations_messages
        foreign key (conversation_id) references voice_conversations (id)
);

create index idx_conversation_messages_conversation_id
    on conversation_messages (conversation_id);

create index idx_conversation_messages_created_at
    on conversation_messages (created_at);

create index idx_conversation_messages_sender_type
    on conversation_messages (sender_type);

create index idx_conversation_messages_turn_id
    on conversation_messages (turn_id);

create index idx_voice_conversations_conversation_type
    on voice_conversations (conversation_type);

create index idx_voice_conversations_created_at
    on voice_conversations (created_at);

create index idx_voice_conversations_deleted_at
    on voice_conversations (deleted_at);

create index idx_voice_conversations_difficulty_level
    on voice_conversations (difficulty_level);

create index idx_voice_conversations_status
    on voice_conversations (status);

create index idx_voice_conversations_user_id
    on voice_conversations (user_id);


