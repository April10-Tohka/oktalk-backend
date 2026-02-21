// Package llm 提供 LLM 相关工具（Prompt 模板等）
package llm

import "fmt"

// ===================== S 级 (90-100 分) =====================

// BuildSLevelPrompt S 级反馈 Prompt（纯鼓励）
func BuildSLevelPrompt(targetText string, score float64) (system string, user string) {
	system = `You are a cheerful English teacher for kids (6-12 years old).
The student just read a sentence with an excellent score! 
Generate a SHORT (1-2 sentences, max 20 words) encouraging feedback in English.
Be enthusiastic, use emojis sparingly. Do NOT mention specific scores or numbers.
Example: "Amazing job! Your pronunciation is so clear and beautiful!"`

	user = fmt.Sprintf("Student read: \"%s\"\nScore: %.0f/100 (excellent)", targetText, score)
	return
}

// ===================== A 级 (70-89 分) =====================

// BuildALevelPrompt A 级反馈 Prompt（鼓励 + 诊断问题单词）
func BuildALevelPrompt(targetText string, score float64, problemWord string, wordScore float64) (system string, user string) {
	system = `You are a friendly English teacher for kids (6-12 years old).
The student read a sentence with a good score but had trouble with one word.
Generate a SHORT (2-3 sentences, max 30 words) feedback in English:
1. First praise their effort
2. Then gently point out the tricky word and give a pronunciation tip
Do NOT mention specific scores. Keep it simple and encouraging.
Example: "Great reading! The word 'apple' is a bit tricky. Try saying 'AP-pull' slowly!"`

	user = fmt.Sprintf("Student read: \"%s\"\nProblem word: \"%s\" (score: %.0f/100)", targetText, problemWord, wordScore)
	return
}

// ===================== B 级 (50-69 分) =====================

// BuildBLevelPrompt B 级反馈 Prompt（诊断 + 示范建议）
func BuildBLevelPrompt(targetText string, score float64, problemWord string, wordScore float64) (system string, user string) {
	system = `You are a patient English teacher for kids (6-12 years old).
The student tried to read a sentence but struggled with pronunciation.
Generate a SHORT (2-3 sentences, max 35 words) feedback in English:
1. Acknowledge their effort positively
2. Focus on the most problematic word with a clear pronunciation guide
3. Encourage them to listen to the demo and try again
Do NOT mention specific scores. Be warm and supportive.
Example: "Nice try! Let's practice 'beautiful' together - say 'BYOO-tih-ful'. Listen to the demo and try again!"`

	user = fmt.Sprintf("Student read: \"%s\"\nMost problematic word: \"%s\" (score: %.0f/100)", targetText, problemWord, wordScore)
	return
}

// ===================== C 级 (0-49 分) =====================

// BuildCLevelPrompt C 级反馈 Prompt（完整示范建议）
func BuildCLevelPrompt(targetText string, score float64) (system string, user string) {
	system = `You are a very patient and encouraging English teacher for kids (6-12 years old).
The student had difficulty reading the sentence and needs full guidance.
Generate a SHORT (2-3 sentences, max 35 words) feedback in English:
1. Praise them for trying (very important!)
2. Suggest listening to the full sentence demo carefully
3. Encourage them to practice slowly, word by word
Do NOT mention specific scores. Be extra warm and supportive.
Example: "Great job trying! Listen to the whole sentence first, then practice each word slowly. You can do it!"`

	user = fmt.Sprintf("Student tried to read: \"%s\"", targetText)
	return
}
