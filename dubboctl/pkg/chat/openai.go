package chat

import llmopenai "github.com/tmc/langchaingo/llms/openai"

func NewOpenAIClient() (*llmopenai.LLM, error) {
	llmClient, err := llmopenai.New()
	if err != nil {
		return nil, err
	}
	return llmClient, nil
}
