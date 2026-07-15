# AI 语音对话 Agent 系统设计方案

> 面向「龙宝 OK」口语学习系统 · 基于 `internal/service/free_talk.go` 与 `internal/service/conversation_memory.go` 现状
> 目标：将"输入→处理→输出"的固定流水线，升级为具备 **LLM + Planning + Tool use + Memory** 四要素的 Agent。

---

## 1. 现状与问题分析

### 1.1 当前数据流（来自 `free_talk.go`）

```
App PCM 音频
   │  rawAudioChan
   ▼
vadSendGoroutine ──gRPC──▶ VAD 服务
                              │ SPEECH_START / SPEECH_END(+PCM)
                              ▼
                        vadRecvGoroutine
                              │ asrAudioChan
                              ▼
                        asrGoroutine (ASR 识别)
                              │ llmInputChan ← 识别文本
                              ▼
                        llmGoroutine (LLM 流式)
                              │ llmOutputChan ← 句子切分后的文本
                              ▼
                        ttsGoroutine (TTS 合成 PCM)
                              │ writeChan
                              ▼
                        writerGoroutine ──▶ App 播放

silenceWatcherGoroutine：AI 说完后 10s 用户未开口 → 注入 silenceTrigger
                              → llmGoroutine 生成"主动引导语"
```

整个链路是一条**线性管道**，每一段 goroutine 只做一件固定的事，靠 channel 串联。

### 1.2 四要素现状对照

| Agent 要素 | 现状 | 问题 |
|---|---|---|
| **LLM（大脑）** | `llmProvider.ChatHistoryStream` 流式对话，已落地 | 仅作为"应答器"：来一句用户话 → 回一句。无决策、无目标驱动。 |
| **Planning（规划）** | 无 | 没有任何任务分解 / 目标管理。唯一"主动行为"是 silenceWatcher 的静默引导（且只是换个 prompt）。 |
| **Tool use（执行）** | 无 | LLM 只能"说"，不能"做"。但系统**其实已经具备大量可执行能力**（见 §1.3），只是没对 LLM 开放。 |
| **Memory（记忆）** | `ConversationMemory`：短期滑动窗口 + 摘要重建 | 仅短期；跨会话不持久；用户画像 `buildUserProfile()` 是**桩**（返回空名 + 默认年龄）；无学习历史/情景记忆。 |

### 1.3 系统已有但"未对 LLM 开放"的能力（天然的 Tool 候选）

- `EvaluationProvider.Assess` —— 发音评测（讯飞），返回总分/分项/单词级结果。
- `PronunciationService` —— 单元列表、开始练习、推进、单元总结（`GetUnitList`/`StartSession`/`Advance`/`GetSummary`）。
- `EvaluateService` —— 评测历史/详情/标准示范音频。
- `report_service.go` —— 学习报告（周报等）。
- 用户服务 / DB —— 用户画像、学习进度、历史记录。

> 关键结论：**能力不缺，缺的是"让 LLM 自主调度这些能力"的编排层与接口。**

### 1.4 核心痛点小结

1. 固定线性流程，缺乏 Planning，AI 永远在"被动接话"。
2. LLM 无工具调用能力（接口层就没定义），无法执行评测/查报告/拉练习。
3. 记忆仅短期且不持久，用户画像为空，AI 每轮都"重新认识"孩子。
4. 角色"龙宝 OK"能力单薄，只会聊天，无法组织一次完整的"练发音→反馈→鼓励"任务。
5. 扩展性差：每加一种能力都要改 `llmGoroutine` 的硬编码逻辑。

---

## 2. Agent 四要素能力映射（目标态）

```
                    ┌──────────────────────────────────────────┐
                    │            Agent 编排层 (新增)              │
                    │                                            │
  用户语音(ASR文本) ─┤  Planning（规划器/目标管理）               │
                    │     │                                      │
                    │     ▼                                      │
                    │  LLM 大脑（决策：直接回 / 调工具 / 多步）   │──┐
                    │     │                    ▲                 │  │
                    │     │ 工具调用            │ 工具结果        │  │
                    │     ▼                    │                 │  │
                    │  Tool use（执行器 + 工具注册表）             │  │
                    │     │                                      │  │
                    │     ├─▶ 发音评测 / 练习单元 / 学习报告 / ...  │  │
                    │                                            │  │
                    │  Memory（记忆层）                           │  │
                    │     ├─ 短期工作记忆（对话窗口）             │  │
                    │     ├─ 长期语义记忆（用户画像/水平）         │◀─┘
                    │     └─ 情景记忆（学习历史/弱项）             │
                    └──────────────────────────────────────────┘
                                    │ 最终文本（流式）
                                    ▼
                              TTS → App 播放
```

| 要素 | 目标态 | 与现状的差距（改造量） |
|---|---|---|
| LLM | 保持 Qwen，但需**支持工具调用**（Function Calling） | 中：给 `LLMProvider` 加工具调用方法 + `QwenAdapter` 实现（SDK 已支持） |
| Planning | Agent Loop（ReAct 风格）驱动多步决策；支持"会话目标" | 大：新增编排层，替换 `llmGoroutine` 内"直接调 LLM"的逻辑 |
| Tool use | 工具注册表 + 同步/异步工具，对 LLM 开放 §1.3 能力 | 中：定义 Tool 抽象 + 把已有 service 包成工具 |
| Memory | 三层记忆（短期/长期/情景），跨会话持久化 | 中：升级 `ConversationMemory`，接入用户服务与 DB 历史 |

---

## 3. 总体架构

在 **ASR 之后、TTS 之前**插入一个 **Agent 编排层（Agent Loop）**，原 `llmGoroutine` 退化为"把用户输入交给 Agent、把 Agent 产出的文本流喂给 TTS"。

```
asrGoroutine ──llmInputChan──▶ AgentLoop（新增 internal/agent 包）
                                     │
                                     │  ① 组装 (system + memory + tools + history)
                                     │  ② 调 LLM（带 tools，流式）
                                     │  ③ 解析：
                                     │       - 文本 → 流式投给 llmOutputChan（TTS）
                                     │       - tool_call → 执行工具 → 结果回灌 LLM
                                     │       - 交互工具 → 让出控制权，等用户音频
                                     │  ④ 循环直到 LLM 产出"终态文本"（无新工具调用）
                                     ▼
                              llmOutputChan（句子切分后）▶ ttsGoroutine（保持不变）
```

**复用原则**：`vadSend/vadRecv`、`asrGoroutine`、`ttsGoroutine`、`writerGoroutine`、`silenceWatcherGoroutine`、句子切分 goroutine **全部保持不变**，只在"LLM 这一环"做 Agent 化替换，降低风险。

---

## 4. 详细设计

### 4.1 LLM 大脑层（需补工具调用能力）

当前 `domain.LLMProvider` 只有 `Chat`/`ChatStream`/`NewConversation`/`ConversationChatStream`/`ChatHistoryStream`，**没有工具调用**。而 `QwenAdapter` 用的是 OpenAI **Responses API**（`responses.ResponseNewParams`），该 API 原生支持 `Tools`（`OfFunction`）。因此：

**改造 1 — domain 层新增工具调用契约**

```go
// domain/llm.go 新增
type ToolSpec struct {
    Name        string             // 工具名，如 "assess_pronunciation"
    Description string
    Parameters  string             // JSON Schema（参数定义）
}

type ToolCall struct {
    ID        string              // 工具调用 id（回灌用）
    Name      string
    Arguments string              // JSON 字符串
}

// 流式工具对话：LLM 可能交替产出 文本片段 与 工具调用
type AgentStreamEvent struct {
    Text   string   // 增量文本（可能为 ""）
    IsDone bool     // 本轮（含工具循环）结束
    ToolCalls []ToolCall  // 本步产生的工具调用（非流式累积，通常一次性给全）
}

// 在 LLMProvider 接口新增：
ChatWithToolsStream(ctx context.Context, req AgentRequest) *ssestream.Stream[AgentStreamEvent]
```

**改造 2 — QwenAdapter 实现**：在 `responses.ResponseNewParams` 上挂 `Tools: []responses.ToolUnionParam{...}`，流式解析 `response.output` 中的 `function_call` 事件，转换为 `AgentStreamEvent`。`enable_thinking:false` 维持低延迟。

> 若暂时不想改 LLM 层，可走 **提示词式 ReAct 降级方案**（让 LLM 输出 `<tool>name{args}</tool>` 文本，后端正则解析）。但原生 function calling 更稳、更省 token，作为毕设**推荐原生方案**。

### 4.2 Planning 规划层（Agent Loop）

核心是 **ReAct 风格循环**：`思考(Thought) → 行动(Act) → 观察(Observation)` 直到给出最终回复。

**单轮 Agent Loop 伪代码：**

```
func (a *Agent) RunTurn(userText string) {
    a.memory.AddUser(userText)
    for step := 0; step < maxSteps; step++ {
        events := a.llm.ChatWithToolsStream(a.buildRequest())
        var textBuilder, pendingToolCalls
        for ev := range events {
            if ev.Text != "" {
                textBuilder.Write(ev.Text)
                a.emitToken(ev.Text)        // 实时流式 → TTS
            }
            pendingToolCalls = append(pendingToolCalls, ev.ToolCalls...)
        }
        if len(pendingToolCalls) == 0 {
            a.memory.AddAssistant(textBuilder.String())
            return                      // 无工具调用 = 本轮结束
        }
        // 执行工具，把结果作为 observation 回灌
        for tc := range pendingToolCalls {
            result := a.registry.Execute(ctx, tc)
            a.memory.AddToolResult(tc.ID, tc.Name, result)
        }
        // 循环继续：LLM 拿到工具结果后决定下一步
    }
}
```

**会话级目标（Goal）管理（规划进阶）**：
- 每次开场由 LLM（或配置）设定一个"本次会话目标"，如 *"带孩子练 3 个动物单词并完成一次评测"*。
- Agent 维护 `GoalState`（已完成步骤 / 待办），在每轮后自我检查是否达成，未达成则主动用 `silenceWatcher` 的触发机制推进（见 §4.5）。
- 这把"被动接话"升级为"带目的的引导式教学"。

### 4.3 Tool use 执行层

**Tool 抽象：**

```go
type Tool interface {
    Spec() domain.ToolSpec
    // Execute 返回可被 LLM 理解的文本结果（语音 Agent 不把原始音频塞给 LLM）
    Execute(ctx context.Context, args string) (string, error)
}

type Registry struct { tools map[string]Tool }
func (r *Registry) Execute(ctx, tc domain.ToolCall) string { ... }
```

**工具分两类（语音场景关键区分）：**

| 类型 | 工具示例 | 特点 |
|---|---|---|
| **同步工具** | `get_user_profile`、`get_learning_report`、`list_pronunciation_units`、`get_unit_progress` | 立即返回数据，LLM 据此调整话题/难度。 |
| **异步交互工具** | `start_pronunciation_practice(word)`、`assess_pronunciation()` | 需要**用户开口说话**，Agent 让出控制权 → TTS 引导 → 等 ASR 音频 → 跑 `EvaluationProvider` → 结果回灌 LLM 继续。 |

**具体工具清单（MVP）：**

1. `list_pronunciation_units(type?)` → 返回可选练习单元（接 `PronunciationService.GetUnitList`）。
2. `start_pronunciation_practice(unit_id, item_index?)` → 取当前练习词/句 + 标准音频 URL（接 `StartSession`/`Advance`）。**交互**：让 AI 先念示范 → 提示孩子跟读。
3. `assess_pronunciation(audio)` → 对孩子刚才的跟读音频调用 `EvaluationProvider.Assess`，返回分数 + 问题词（接 `EvaluationProvider`）。**注意**：音频由 `asrGoroutine` 侧录到的"跟读句 PCM"提供，不是文本。
4. `get_learning_report(range)` → 学习报告摘要（接 `report_service`）。
5. `get_user_profile()` / `update_user_profile()` → 长期画像读写（接用户服务，替换当前桩）。
6. `end_turn()`（元工具）→ 显式结束本轮（可选，帮助流式收尾）。

> 工具描述（Description）要写得"像给龙宝 OK 的指令"，让它知道何时该用哪个工具。例如：`assess_pronunciation` 的描述要强调"只有在孩子刚跟读了一句之后调用"。

### 4.4 Memory 记忆层（三层）

```
                 ┌─────────────────────────────┐
   当前对话 ───▶  │ ① 短期工作记忆（对话窗口）   │  ← 现有 ConversationMemory 升级
                 │   近期轮次 + 摘要重建         │
                 └──────────────┬──────────────┘
                                │ 抽取
                 ┌──────────────▼──────────────┐
   会话结束 ───▶  │ ② 长期语义记忆（用户画像）   │  ← 持久化到用户服务/DB
                 │   姓名/年龄/兴趣/英语水平     │     （替换 buildUserProfile 桩）
                 └──────────────┬──────────────┘
                                │ 记录
                 ┌──────────────▼──────────────┐
   每次练习 ───▶  │ ③ 情景记忆（学习历史）       │  ← 已有 DB：PronunciationRecord /
                 │   弱项词/得分趋势/常错音素    │     LearningReport，经工具查询
                 └─────────────────────────────┘
```

- **① 短期**：保留 `ConversationMemory` 的滑动窗口 + 摘要重建，但改为 `AgentMemory`，额外存 tool_call/observation 轨迹（供 LLM 回看"我刚做了什么"）。
- **② 长期语义**：`buildUserProfile()` 当前返回空，需接用户服务持久化（P2 阶段）。LLM 开场先 `get_user_profile` 拿到姓名/年龄/水平，个性化语气与难度。
- **③ 情景**：发音记录、报告已在 DB。通过 §4.3 的 `get_learning_report` 等工具，让 Agent 能说"你上周 'th' 音老错，今天我们专门练一下~"。

### 4.5 语音场景的特殊设计（重点！）

文本 Agent 的工具调用"秒回"，但**语音对话对延迟极度敏感**，且部分工具需要"等孩子说话"。必须处理：

1. **边想边说（Verbalize while planning）**
   LLM 在决定调工具前，先流式产出一句"口头过渡"，如 *"让我看看你上次的练习~"*，再执行 `get_learning_report`。用户体验上"AI 在思考但没冷场"。这通过在 system prompt 里约束"调工具前先用一句话过渡"实现，且过渡句直接进 TTS。

2. **交互式工具 = 控制权让渡**
   当 LLM 调用 `start_pronunciation_practice` / `assess_pronunciation` 时，Agent Loop **暂停循环**，向 `ttsGoroutine` 发引导语（"来，跟着我说：apple"），并把 `asrGoroutine` 的下一句音频"截留"为工具输入（而非普通对话），跑完 `EvaluationProvider` 后把结果作为 observation 回灌 LLM，继续循环。本质是**在对话流里嵌入一个"练习子流程"**。

3. **静默/主动触发升级**
   现有 `silenceWatcherGoroutine` 注入 `silenceTrigger` 只是换个引导 prompt。升级后：静默超时 → Agent 检查 `GoalState`，若"本次目标未完成"则主动发起一个工具动作（如"我们刚才没练完那个单词哦，再来一次？"），把被动引导变成主动推进任务。

4. **复用现有流式基建**
   句子切分 goroutine、`ttsNewTurnChan`/`llmOutputChan`/`MsgTypeLLMToken`/`MsgTypeTurnEnd` 全部复用，Agent 产出的文本仍按原有方式流式合成语音，前端零改动。

---

## 5. 与现有代码的改造点（落地清单）

| 文件 | 改造 |
|---|---|
| `internal/domain/llm.go` | 新增 `ToolSpec`/`ToolCall`/`AgentStreamEvent`/`ChatWithToolsStream`，扩展 `LLMProvider` 接口。 |
| `internal/infrastructure/llm/qwen/adapter.go` | 实现 `ChatWithToolsStream`：用 Responses API `Tools` 字段 + 流式解析 function_call。 |
| `internal/service/free_talk.go` | `llmGoroutine` 内的"直接调 `ChatHistoryStream`"替换为"委托 `agent.RunTurn`"；`silenceWatcherGoroutine` 的触发改为调 Agent 的主动规划入口。其余 goroutine 不变。 |
| `internal/service/conversation_memory.go` | 升级为 `AgentMemory`：保留短期窗口/摘要；新增 tool 轨迹存储；提供长期/情景记忆查询钩子。 |
| `internal/service/buildUserProfile`（桩） | 接用户服务，持久化真实画像。 |
| **新增** `internal/agent/` | `agent.go`（Loop+目标）、`tools.go`（Tool 接口）、`registry.go`、`memory.go`、`tools_*.go`（各工具实现）、`prompt.go`（龙宝 OK 人格 + 工具使用约定）。 |

---

## 6. 分阶段实施路线

- **P0（接口与骨架）**：`domain.LLMProvider` 加工具调用契约；`QwenAdapter` 实现；定义 `Tool`/`Registry`；`internal/agent` 包空跑（先只透传文本，验证不破坏现有对话）。
- **P1（最小可用 Agent）**：接入 2~3 个**同步工具**（`get_user_profile`、`list_pronunciation_units`、`get_learning_report`）；单轮 ReAct Loop；LLM 能"查了再聊"。
- **P2（记忆增强）**：长期语义记忆（用户画像持久化）+ 情景记忆（学习历史经工具可读）；`AgentMemory` 替代 `ConversationMemory`。
- **P3（交互式工具 + 主动规划）**：`start_pronunciation_practice` / `assess_pronunciation` 子流程；`silenceWatcher` 升级为基于 `GoalState` 的主动推进；龙宝 OK 能组织一次完整"示范→跟读→评测→鼓励"。
- **P4（打磨）**：多轮目标规划、工具调用预算/超时防护、儿童安全约束（工具白名单、禁止联网/外部副作用）、评测与延迟优化。

---

## 7. 风险与权衡

1. **延迟 vs 能力**：多步工具调用会增加轮次延迟。缓解：① 过渡句先播（§4.5.1）；② 同步工具并行/缓存；③ 限制 `maxSteps`（如 4）。
2. **儿童安全**：Agent 只能调用**白名单工具**，严禁任意代码/联网/外部副作用；工具参数做校验（如 `unit_id` 必须存在）。
3. **成本**：工具轨迹占 token。缓解：短期记忆做摘要；工具结果只回灌"精炼文本"而非原始音频/XML。
4. **流式下工具调用的工程复杂度**：需在流式事件中区分"文本"与"tool_call"，并管理"等待用户音频"的异步态。建议 P0 先用非流式验证 Loop 正确性，再接回流式。
5. **可观测性**：记录每轮的 thought/action/observation 轨迹，便于毕设演示与调试。

---

## 8. 小结

当前系统已具备优秀的**实时语音管道**（VAD/ASR/LLM/TTS 流式串联）与丰富的**底层能力**（发音评测、练习单元、报告），但 LLM 只是"应答器"，缺 Planning、Tool use、持久 Memory。

本方案的核心思路是：**不动语音管道，只在 ASR 与 TTS 之间插入一个 Agent 编排层**，补齐四要素：
- 给 LLM 加**工具调用接口**（补大脑的执行出口）；
- 引入 **ReAct 式 Agent Loop + 会话目标**（补规划）；
- 把已有 service 包成**同步/交互式工具**（补执行）；
- 建**三层记忆**并持久化用户画像与学习历史（补记忆）。

最终让"龙宝 OK"从一个"陪聊伙伴"升级为"会规划、能评测、记得住每个孩子的陪伴式 AI 教练"。
