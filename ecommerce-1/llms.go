package main

import (
	"fmt"
	"log"

	"github.com/tmc/langchaingo/llms"
	_ "github.com/tmc/langchaingo/llms/anthropic"
	_ "github.com/tmc/langchaingo/llms/cohere"
	_ "github.com/tmc/langchaingo/llms/googleai"
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

const ServiceModelSeperator = " : "

type LLMClient struct {
	Name                   string
	Model                  string
	WeightsAccess          WeightsAccessType
	NumParameters          string
	InputContextWindowSize int
	Instance               llms.Model
}

type LLMClientsMap map[string]*LLMClient

// Simple wrapper to get a specific named LLM initializer for whatever mischief we get up to with it.
func getLLMClient(name, model string, clientsMap LLMClientsMap) *LLMClient {
	key := fmt.Sprintf("%s%s%s", name, ServiceModelSeperator, model)
	fmt.Printf("- Looking up key %s\n", key)
	client := clientsMap[key]
	if client == nil {
		log.Fatalf("Error: could not find client for %s", key)
	}
	fmt.Printf("- Found client %v\n", client)
	return clientsMap[key]
}

func initialiseLLMClients(localServerUrl string) map[string]*LLMClient {

	clients := []LLMClient{
		{
			Name: "Ollama/OpenAI", Model: "llama3", WeightsAccess: Open, NumParameters: "8b", InputContextWindowSize: 8192,
			Instance: func() llms.Model {
				model, err := openai.New(openai.WithModel("llama3:instruct"), openai.WithBaseURL(localServerUrl))
				if err != nil {
					log.Printf("Error initializing model %s: %v", "llama3", err)
					return nil
				}
				return model
			}(),
		},
		// {
		// 	Name: "Groq", Model: "llama3-8b-8192", WeightsAccess: Open, NumParameters: "8b", InputContextWindowSize: 8192,
		// 	Instance: func() llms.Model {
		// 		model, err := openai.New(
		// 			openai.WithModel("llama3-8b-8192"),
		// 			openai.WithBaseURL("https://api.groq.com/openai/v1"),
		// 			openai.WithToken(os.Getenv("GROQ_API_KEY")),
		// 		)
		// 		if err != nil {
		// 			log.Printf("Error initializing model %s: %v", "llama3-8b-8192", err)
		// 			return nil
		// 		}
		// 		return model
		// 	}(),
		// },
		// {
		// 	Name: "Groq", Model: "llama3-70b-8192", WeightsAccess: Open, NumParameters: "70b", InputContextWindowSize: 8192,
		// 	Instance: func() llms.Model {
		// 		model, err := openai.New(
		// 			openai.WithModel("llama3-70b-8192"),
		// 			openai.WithBaseURL("https://api.groq.com/openai/v1"),
		// 			openai.WithToken(os.Getenv("GROQ_API_KEY")),
		// 		)
		// 		if err != nil {
		// 			log.Printf("Error initializing model %s: %v", "llama3-70b-8192", err)
		// 			return nil
		// 		}
		// 		return model
		// 	}(),
		// },
		// {
		// 	Name: "Mistral", Model: "llama3-70b-8192", WeightsAccess: Open, NumParameters: "70b", InputContextWindowSize: 8192,
		// 	Instance: func() llms.Model {
		// 		model, err := openai.New(
		// 			openai.WithModel("llama3-70b-8192"),
		// 			openai.WithBaseURL("TODO"),
		// 			openai.WithToken(os.Getenv("MISTRAL_API_KEY")),
		// 		)
		// 		if err != nil {
		// 			log.Printf("Error initializing model %s: %v", "llama3-70b-8192", err)
		// 			return nil
		// 		}
		// 		return model
		// 	}(),
		// },
		// {
		// 	Name: "Ollama/OpenAI", Model: "codestral-22B-v0.1", WeightsAccess: Open, NumParameters: "22b", InputContextWindowSize: 32768,
		// 	Instance: func() llms.Model {
		// 		model, err := openai.New(openai.WithModel("codestral"), openai.WithBaseURL(localServerUrl))
		// 		if err != nil {
		// 			log.Printf("Error initializing model %s: %v", "codestral-22B-v0.1", err)
		// 			return nil
		// 		}
		// 		return model
		// 	}(),
		// },
		// {
		// 	Name: "Ollama/OpenAI", Model: "qwen2:0.5b", WeightsAccess: Open, NumParameters: "0.5b", InputContextWindowSize: 32768,
		// 	Instance: func() llms.Model {
		// 		model, err := openai.New(openai.WithModel("qwen2:0.5b"), openai.WithBaseURL(localServerUrl))
		// 		if err != nil {
		// 			log.Printf("Error initializing model %s: %v", "qwen2:0.5b", err)
		// 			return nil
		// 		}
		// 		return model
		// 	}(),
		// },
		{
			Name: "Ollama/OpenAI", Model: "qwen2:1.5b", WeightsAccess: Open, NumParameters: "1.5b", InputContextWindowSize: 32768,
			Instance: func() llms.Model {
				model, err := openai.New(openai.WithModel("qwen2:1.5b"), openai.WithBaseURL(localServerUrl))
				if err != nil {
					log.Printf("Error initializing model %s: %v", "qwen2:1.5b", err)
					return nil
				}
				return model
			}(),
		},
		// {
		// 	Name: "Ollama/OpenAI", Model: "qwen2:7b", WeightsAccess: Open, NumParameters: "7b", InputContextWindowSize: 131072,
		// 	Instance: func() llms.Model {
		// 		model, err := openai.New(openai.WithModel("qwen2:7b"), openai.WithBaseURL(localServerUrl))
		// 		if err != nil {
		// 			log.Printf("Error initializing model %s: %v", "qwen2:7b", err)
		// 			return nil
		// 		}
		// 		return model
		// 	}(),
		// },
		// {
		// 	Name: "Ollama/OpenAI", Model: "phi3:mini", WeightsAccess: Open, NumParameters: "3.8b", InputContextWindowSize: 4096,
		// 	Instance: func() llms.Model {
		// 		model, err := openai.New(openai.WithModel("phi3:mini"), openai.WithBaseURL(localServerUrl))
		// 		if err != nil {
		// 			log.Printf("Error initializing model %s: %v", "phi3:mini", err)
		// 			return nil
		// 		}
		// 		return model
		// 	}(),
		// },
		// {
		// 	Name: "Ollama/OpenAI", Model: "phi3:medium", WeightsAccess: Open, NumParameters: "14b", InputContextWindowSize: 4096,
		// 	Instance: func() llms.Model {
		// 		model, err := openai.New(openai.WithModel("phi3:medium"), openai.WithBaseURL(localServerUrl))
		// 		if err != nil {
		// 			log.Printf("Error initializing model %s: %v", "phi3:medium", err)
		// 			return nil
		// 		}
		// 		return model
		// 	}(),
		// },
		// {
		// 	Name: "Ollama/OpenAI", Model: "phi3:medium-128k", WeightsAccess: Open, NumParameters: "14b", InputContextWindowSize: 131072,
		// 	Instance: func() llms.Model {
		// 		model, err := openai.New(openai.WithModel("phi3:medium-128k"), openai.WithBaseURL(localServerUrl))
		// 		if err != nil {
		// 			log.Printf("Error initializing model %s: %v", "phi3:medium-128k", err)
		// 			return nil
		// 		}
		// 		return model
		// 	}(),
		// },
		// {
		// 	Name: "Cohere", Model: "Command-R+", WeightsAccess: Open, NumParameters: "104b", InputContextWindowSize: 131072,
		// 	Instance: func() llms.Model {
		// 		model, err := cohere.New(
		// 			cohere.WithModel("command-r-plus"),
		// 			cohere.WithToken(os.Getenv("COHERE_API_KEY")),
		// 		)
		// 		if err != nil {
		// 			log.Printf("Error initializing model %s: %v", "Command-R+", err)
		// 			return nil
		// 		}
		// 		return model
		// 	}(),
		// },
		// {
		// 	Name: "Anthropic", Model: "claude-3-haiku-20240307", WeightsAccess: Closed, NumParameters: "?", InputContextWindowSize: 4096,
		// 	Instance: func() llms.Model {
		// 		model, err := anthropic.New(anthropic.WithModel("claude-3-haiku-20240307"))
		// 		if err != nil {
		// 			log.Printf("Error initializing model %s: %v", "claude-3-haiku-20240307", err)
		// 			return nil
		// 		}
		// 		return model
		// 	}(),
		// },
		// {
		// 	Name: "Anthropic", Model: "claude-3-sonnet-20240229", WeightsAccess: Closed, NumParameters: "?", InputContextWindowSize: 4096,
		// 	Instance: func() llms.Model {
		// 		model, err := anthropic.New(anthropic.WithModel("claude-3-sonnet-20240229"))
		// 		if err != nil {
		// 			log.Printf("Error initializing model %s: %v", "claude-3-sonnet-20240229", err)
		// 			return nil
		// 		}
		// 		return model
		// 	}(),
		// },
		// {
		// 	Name: "Google AI", Model: "Gemini Flash 1.5", WeightsAccess: Closed, NumParameters: "?", InputContextWindowSize: 1048576,
		// 	Instance: func() llms.Model {
		// 		apiKey := os.Getenv("GEMINI_API_KEY")
		// 		model, err := googleai.New(context.Background(),
		// 			googleai.WithAPIKey(apiKey),
		// 			googleai.WithDefaultModel("gemini-1.5-flash-001"))
		// 		if err != nil {
		// 			log.Printf("Error initializing model %s: %v", "Gemini Flash 1.5", err)
		// 			return nil
		// 		}
		// 		return model
		// 	}(),
		// },
		// {
		// 	Name: "HuggingFace", Model: "Meta-Llama-3-8B", WeightsAccess: Open, NumParameters: "8b", InputContextWindowSize: 8192,
		// 	Instance: func() llms.Model {
		// 		model, err := huggingface.New(huggingface.WithModel("Meta-Llama-3-8B"))
		// 		if err != nil {
		// 			log.Printf("Error initializing model %s: %v", "Meta-Llama-3-8B", err)
		// 			return nil
		// 		}
		// 		return model
		// 	}(),
		// },
		// {
		// 	Name: "Llamafile", Model: "open-mistral-7b", WeightsAccess: Open, NumParameters: "7b", InputContextWindowSize: 8192,
		// 	Instance: func() llms.Model {
		// 		options := []llamafile.Option{
		// 			llamafile.WithEmbeddingSize(2048),
		// 			llamafile.WithTemperature(0.8),
		// 		}
		// 		model, err := llamafile.New(options...)
		// 		if err != nil {
		// 			log.Printf("Error initializing model %s: %v", "open-mistral-7b", err)
		// 			return nil
		// 		}
		// 		return model
		// 	}(),
		// },
		// {
		// 	Name: "Ollama", Model: "llama3:instruct", WeightsAccess: Open, NumParameters: "8b", InputContextWindowSize: 8192,
		// 	Instance: func() llms.Model {
		// 		model, err := ollama.New(ollama.WithModel("llama3:instruct"))
		// 		if err != nil {
		// 			log.Printf("Error initializing model %s: %v", "llama3:instruct", err)
		// 			return nil
		// 		}
		// 		return model
		// 	}(),
		// },
		// {
		// 	Name: "OpenAI GPT-4-turbo-preview", Model: "gpt-4-turbo-preview", WeightsAccess: Open, NumParameters: "?", InputContextWindowSize: 8192,
		// 	Instance: func() llms.Model {
		// 		model, err := openai.New(openai.WithModel("gpt-4-turbo-preview"))
		// 		if err != nil {
		// 			log.Printf("Error initializing model %s: %v", "gpt-4-turbo-preview", err)
		// 			return nil
		// 		}
		// 		return model
		// 	}(),
		// },
	}

	// convert them to a map for easy access
	clientsMap := make(LLMClientsMap)
	for _, client := range clients {
		key := client.Name + ServiceModelSeperator + client.Model
		clientsMap[key] = &client
	}
	return clientsMap
}
