package qwen

import (
	"fmt"
	"strings"
)

// PromptTemplate Prompt 模板
type PromptTemplate struct {
	Name     string
	Template string
	Vars     []string
}

// PromptManager Prompt 管理器
type PromptManager struct {
	templates map[string]*PromptTemplate
}

// NewPromptManager 创建 Prompt 管理器
func NewPromptManager() *PromptManager {
	pm := &PromptManager{
		templates: make(map[string]*PromptTemplate),
	}
	pm.registerDefaultTemplates()
	return pm
}

// registerDefaultTemplates 注册默认模板
func (pm *PromptManager) registerDefaultTemplates() {
	// 发音反馈模板
	pm.Register(&PromptTemplate{
		Name: "pronunciation_feedback",
		Template: `你是一位专业的英语发音教练。根据以下评测结果，为用户提供友好、鼓励性的反馈和改进建议。

评测文本: {{text}}
综合评分: {{score}}
准确度: {{accuracy}}
流利度: {{fluency}}
完整度: {{completeness}}
主要问题: {{problems}}

请提供：
1. 简短的总体评价（1-2句话）
2. 具体的发音问题分析
3. 针对性的改进建议
4. 鼓励性的结语

请用简洁友好的语气回复。`,
		Vars: []string{"text", "score", "accuracy", "fluency", "completeness", "problems"},
	})

	// 对话模板
	pm.Register(&PromptTemplate{
		Name: "conversation",
		Template: `你是一位友好的英语口语练习伙伴。场景：{{scenario}}。

请根据对话历史继续与用户进行自然的英语对话。注意：
1. 使用适合用户水平的词汇和句式
2. 在对话中自然地纠正用户的表达错误
3. 保持对话有趣且有教育意义

用户水平: {{level}}
对话历史:
{{history}}`,
		Vars: []string{"scenario", "level", "history"},
	})

	// 报告生成模板
	pm.Register(&PromptTemplate{
		Name: "report_generation",
		Template: `你是一位专业的英语学习分析师。请根据以下数据生成一份学习报告。

时间范围: {{period}}
评测次数: {{count}}
平均分数: {{avg_score}}
最高分数: {{max_score}}
常见问题: {{common_problems}}

请生成一份包含以下内容的报告：
1. 学习进度总结
2. 优势和进步
3. 需要改进的方面
4. 具体的练习建议
5. 鼓励性的结语`,
		Vars: []string{"period", "count", "avg_score", "max_score", "common_problems"},
	})
}

// Register 注册模板
func (pm *PromptManager) Register(template *PromptTemplate) {
	pm.templates[template.Name] = template
}

// Get 获取模板
func (pm *PromptManager) Get(name string) *PromptTemplate {
	return pm.templates[name]
}

// Build 构建 Prompt
func (pm *PromptManager) Build(name string, vars map[string]string) (string, error) {
	template, ok := pm.templates[name]
	if !ok {
		return "", fmt.Errorf("template not found: %s", name)
	}

	result := template.Template
	for key, value := range vars {
		placeholder := fmt.Sprintf("{{%s}}", key)
		result = strings.ReplaceAll(result, placeholder, value)
	}

	return result, nil
}
