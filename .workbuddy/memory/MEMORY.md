# 长期项目记忆：OKtalk-backend1（龙宝 OK 英语陪练 Agent）

## 架构关键不变量（踩坑必记）

- **Agent ReAct 循环依赖 `IsDone` 终态事件判定单步结束**。`qwen/adapter.go` 的
  `agentStreamDecoder` **任何一轮（无论有无工具调用）都必须在流末尾产出 `IsDone:true`
  事件**，否则上层 `agent.reactLoopDecoder` 无法终止单步 → 纯文本轮会无限重试调用 LLM。
  （已在 P1 修复并加 `adapter_test.go` 回归测试，改动此处务必先跑该测试。）
- **`constant.FunctionCall` / `constant.FunctionCallOutput` 是具名 string 类型**
  （来自 openai-go `shared/constant`），赋值时必须写 `constant.FunctionCall("function_call")`
  这类"类型(字面量)"形式，不能当普通值/常量用。
- **工具实现放在 `service` 包而非 `agent` 包**：`agent` 不能反向依赖 `service`，避免循环引用；
  工具通过 `Session.registerAgentTools` 在运行时注入注册表，按"依赖非空"原则 nil-safe 注册。
- **异步交互工具靠"控制权让渡"机制（P2）**：`start_pronunciation_practice`/`assess_pronunciation`
  调 `s.armFollowRead(...)` 置 `awaitingFollowRead=true`；`asrGoroutine` 每句 ASR 成功后先
  `consumeFollowRead()`，命中则截留该句音频跑 `evalProvider.Assess`，把结果作为 observation 回灌
  `llmInputChan`（**不当作普通对话**）。`evalProvider==nil` 或评测失败时退化为普通对话（已清 awaiting）。
  所有练习态字段（`awaitingFollowRead`/`practiceTargetText`/`practiceCategory`/`practiceSessionID`/
  `practiceItemIndex`）必须由 `practiceMu` 守卫。改此处务必跑 `internal/service` 单测。
- **Registry 稳定性/安全三防护（P4，在 `internal/agent/tools.go`）**：`Execute` 顺序执行
  ① 儿童安全白名单拦截（`allowList` 非 nil 时拒绝名单外工具名并告警）② 跨会话工具调用预算
  （`budget`/`used`，达上限返回 "budget exhausted" 提示强制本轮收尾）③ 单次工具执行超时
  （`toolTimeout`，`context.WithTimeout` 包裹 `t.Execute`）。`agent.reactLoopDecoder` 额外对每步 LLM
  调用包裹 `turnTimeout=2min`；其 `stepCancel` 必须在 `endStep()` 释放，**绝不能 `defer`**
  （defer 会在本步首个文本事件后误取消 context，破坏后续流式读取）。改此处务必跑 `internal/agent` 单测（`registry_test.go`）。
- **会话级目标 `GoalState`（P3，在 `free_talk.go` 的 Session）**：`goalState *GoalState` + `goalMu`
  守卫；`setGoal`/`updateGoalProgress`/`goalContextNote` 三方法。`goalContextNote()` 在 `llmGoroutine`
  每轮（含静默触发）被注入 messages——**必须复制切片**（因 `memory.Messages()` 返回内部底层数组，
  直接 append 会污染持久化历史）。目标未完成为 LLM 注入"主动推进下一步"指令，静默超时即驱动主动引导。
  `set_session_goal`/`update_goal_progress` 两个工具在 `registerAgentTools` 中**始终注册**（无外部依赖）。
- **儿童安全 System Prompt 加固（P4，在 `conversation_memory.go` `buildSystemPrompt`）**：新增
  "Child-safety guardrails" 小节，约束仅用白名单工具、不联网/不执行代码/不泄露个人信息。
  `start_pronunciation_practice` 增加 `unit_id` 存在性校验（无效返回安全提示，不报错）。
- **长期记忆用独立 `AgentProfileStore`（Redis+内存兜底，P2），不复用 `model.UserProfile`**：
  避免扰动由 UserService/AuthService 管理的共享表与 DB 迁移。更新画像走 `update_user_profile` 工具
  增量改 `s.userProfile` 值并 `saveUserProfile()`；`ConversationMemory` 现持有 `userProfile *UserProfile`
  （指针）以让改动实时反映进 System Prompt。`NewAgentProfileStore(client)` 在 client==nil 时返回内存实现。
- **⚠️ 长期记忆"写"依赖 LLM 主动调 `update_user_profile`，模型可能"假装记住"而不调工具**（联调实测）：
  qwen 在弱提示下会回复"我记住啦"但**不实际调用工具**，导致记忆静默未持久化（store 为空）。
  务必在 System Prompt 强约束"先调工具再回应，否则等于欺骗小朋友"，或加一层"从用户消息确定性抽取画像字段"的兜底，
  否则跨会话记忆会静默失效。`TestE2E_P1_ToolCallByLLM` 用强约束提示已稳定触发写工具。
- **⚠️ 多轮目标同样依赖 LLM 主动调 `set_session_goal`/`update_goal_progress`，模型也可能"假装规划"不调工具**（P3 实测）。
  `TestE2E_P3_GoalPlanningByLLM` 用强约束提示（"先调工具再回应，否则等于欺骗小朋友"）稳定触发；生产 System Prompt 务必延续此强约束。
  注意：真实 LLM 每轮引入的练习词不固定（dog/cat/...），联调测试第 2 轮须用"我刚才跟着你读的那个词"等上下文引用，避免硬编码单词导致偶发不更新。

## Agent 四要素落地进度（毕设）

- LLM：`domain.LLMProvider.ChatWithToolsStream` + `QwenAdapter`（Responses API，已含 Tools）。
- Planning：P1 起 `reactLoopDecoder` 驱动 ReAct 多步循环（maxSteps=4 防护）；P3 起新增
  `GoalState` 多轮目标规划——`goalContextNote()` 每轮注入目标上下文，静默超时/走神时驱动 LLM 主动推进
  未完成目标（设计文档 §4.5.3）。
- Tool use：现共 8 个工具——3 同步（get_user_profile / list_pronunciation_units /
  get_learning_report）+ 2 异步交互（start_pronunciation_practice / assess_pronunciation，靠"控制权让渡"
  等用户跟读音频）+ 2 长期记忆（get/update_user_profile）+ 2 目标管理（set_session_goal /
  update_goal_progress）。`Run()` 中统一 `registry.SetToolTimeout(15s)` / `SetBudget(50)` /
  `AllowTools(...8个安全工具名)` 加固。
- Memory：P0 仅 `conversation_memory.go` 短期记忆 + 摘要重建；P2 起 `agent_profile_store.go`
  + `userProfile` 镜像 + get/update_user_profile 工具实现跨会话语义记忆（Redis 7天 TTL / 内存兜底）；
  情景记忆仍靠 get_learning_report（ReportService）。
- 语音管道（VAD+ASR+LLM+TTS 多 goroutine）未改动，Agent 仅插在 ASR 与 TTS 之间。

## 验证约定

- `go build ./...` 应全绿。`go vet ./...` 已知有一处预先存在的
  `internal/infrastructure/tts/aliyun/client_test.go:137 unreachable code`（非本次改动，未动）。
- `TestQwenAdapterChatStream` 是真实 API 集成测试，网络/限流偶发 flaky，单独跑通过，与逻辑改动无关。
