package main

import (
	"context"
	"os"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/anthropic"
	"github.com/tmc/langchaingo/llms/cohere"
	"github.com/tmc/langchaingo/llms/googleai"
	_ "github.com/tmc/langchaingo/llms/huggingface"
	_ "github.com/tmc/langchaingo/llms/llamafile"
	_ "github.com/tmc/langchaingo/llms/mistral"
	_ "github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/llms/openai"
)

type WeightsAccessType int

const (
	Closed WeightsAccessType = iota
	Open
)

type NamedLLMInitializer struct {
	Name                   string
	Model                  string
	InitFunc               func() (llms.Model, error)
	WeightsAccess          WeightsAccessType
	NumParameters          string
	InputContextWindowSize int
}

func getLLMs(localServerUrl string) []NamedLLMInitializer {

	//ctx := context.Background()

	llmInitializers := []NamedLLMInitializer{

		// https://huggingface.co/meta-llama/Meta-Llama-3-8B
		{Name: "Ollama/OpenAI", Model: "llama3", WeightsAccess: Open, NumParameters: "8b", InputContextWindowSize: 8192, InitFunc: func() (llms.Model, error) {
			return openai.New(openai.WithModel("llama3:instruct"), openai.WithBaseURL(localServerUrl))
		}},

		{Name: "Groq", Model: "llama3-8b-8192", WeightsAccess: Open, NumParameters: "8b", InputContextWindowSize: 8192, InitFunc: func() (llms.Model, error) {
			return openai.New(
				openai.WithModel("llama3-8b-8192"),
				openai.WithBaseURL("https://api.groq.com/openai/v1"),
				openai.WithToken(os.Getenv("GROQ_API_KEY")),
			)
		}},

		// https://mistral.ai/news/codestral/
		{Name: "Ollama/OpenAI", Model: "codestral-22B-v0.1", WeightsAccess: Open, NumParameters: "22b", InputContextWindowSize: 32768, InitFunc: func() (llms.Model, error) {
			return openai.New(openai.WithModel("codestral"), openai.WithBaseURL(localServerUrl))
		}},

		// https://ollama.com/library/phi3
		{Name: "Ollama/OpenAI", Model: "phi3:mini", WeightsAccess: Open, NumParameters: "3.8b", InputContextWindowSize: 4096, InitFunc: func() (llms.Model, error) {
			return openai.New(openai.WithModel("phi3:mini"), openai.WithBaseURL(localServerUrl))
		}},
		// https://ollama.com/library/phi3
		{Name: "Ollama/OpenAI", Model: "phi3:medium", WeightsAccess: Open, NumParameters: "14b", InputContextWindowSize: 4096, InitFunc: func() (llms.Model, error) {
			return openai.New(openai.WithModel("phi3:medium"), openai.WithBaseURL(localServerUrl))
		}},
		// https://ollama.com/library/phi3
		{Name: "Ollama/OpenAI", Model: "phi3:medium-128k", WeightsAccess: Open, NumParameters: "14b", InputContextWindowSize: 131072, InitFunc: func() (llms.Model, error) {
			return openai.New(openai.WithModel("phi3:medium-128k"), openai.WithBaseURL(localServerUrl))
		}},

		// https://docs.cohere.com/docs/models
		// looking for environment variable COHERE_API_KEY via coherellm_option.go/tokenEnvVarName
		{Name: "Cohere", Model: "Command-R+", WeightsAccess: Open, NumParameters: "104b", InputContextWindowSize: 131072, InitFunc: func() (llms.Model, error) {
			return cohere.New(
				cohere.WithModel("command-r-plus"),
				cohere.WithToken(os.Getenv("COHERE_API_KEY")),
			)
		}},

		{Name: "Groq", Model: "llama3-70b-8192", WeightsAccess: Open, NumParameters: "70b", InputContextWindowSize: 8192, InitFunc: func() (llms.Model, error) {
			return openai.New(openai.WithModel("llama3-70b-8192"), openai.WithBaseURL("https://api.groq.com/openai/v1"))
		}},

		// https://docs.anthropic.com/en/docs/models-overview
		// environment variable required to be set: ANTHROPIC_API_KEY
		{Name: "Anthropic", Model: "claude-3-haiku-20240307", WeightsAccess: Closed, NumParameters: "?", InputContextWindowSize: 4096, InitFunc: func() (llms.Model, error) {
			return anthropic.New(anthropic.WithModel("claude-3-haiku-20240307"))
		}},

		// https://docs.anthropic.com/en/docs/models-overview
		// environment variable required to be set: ANTHROPIC_API_KEY
		{Name: "Anthropic", Model: "claude-3-sonnet-20240229", WeightsAccess: Closed, NumParameters: "?", InputContextWindowSize: 4096, InitFunc: func() (llms.Model, error) {
			return anthropic.New(anthropic.WithModel("claude-3-sonnet-20240229"))
		}},

		// https://cloud.google.com/vertex-ai/generative-ai/docs/learn/model-versioning
		{Name: "Google AI", Model: "Gemini Flash 1.5", WeightsAccess: Closed, NumParameters: "?", InputContextWindowSize: 1048576, InitFunc: func() (llms.Model, error) {
			apiKey := os.Getenv("GEMINI_API_KEY")
			return googleai.New(context.Background(),
				googleai.WithAPIKey(apiKey),
				googleai.WithDefaultModel("gemini-1.5-flash-001"))
		}},

		// https://github.com/meta-llama/llama3/blob/main/MODEL_CARD.md
		// TODO {Name: "HuggingFace", InitFunc: func() (llms.Model, error) {
		// 	return huggingface.New()
		// }},

		// TODO {Name: "Llamafile", InitFunc: func() (llms.Model, error) {
		// 	options := []llamafile.Option{
		// 		llamafile.WithEmbeddingSize(2048),
		// 		llamafile.WithTemperature(0.8),
		// 	}
		// 	return llamafile.New(options...)
		// }},
		// {Name: "Mistral", InitFunc: func() (llms.Model, error) {
		// 	return mistral.New(mistral.WithModel("open-mistral-7b"))
		// }},

		// TODO {Name: "Ollama", InitFunc: func() (llms.Model, error) {
		// 	return ollama.New(ollama.WithModel("llama3:instruct"))
		// }},

		// // https://platform.openai.com/docs/models/gpt-4-turbo-and-gpt-4
		{Name: "OpenAI GPT-4-turbo-preview", InitFunc: func() (llms.Model, error) {
			return openai.New(openai.WithModel("gpt-4-turbo-preview"))
		}},
	}

	return llmInitializers
}
