/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package manager

import (
	"context"
	"log/slog"
	"os"
	"sync"

	"dubbo-admin-ai/config"
	"dubbo-admin-ai/plugins/dashscope"
	"dubbo-admin-ai/plugins/siliconflow"

	"github.com/dusted-go/logging/prettylog"
	"github.com/firebase/genkit/go/core/api"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/googlegenai"
	"github.com/firebase/genkit/go/plugins/pinecone"
	"github.com/joho/godotenv"
)

var (
	gloRegistry *genkit.Genkit
	gloLogger   *slog.Logger
	once        sync.Once
)

func Registry(modelName string, envPath string, logger *slog.Logger) (registry *genkit.Genkit) {
	once.Do(func() {
		gloLogger = logger
		if logger == nil {
			gloLogger = ProductionLogger()
		}
		LoadEnvVars2Config(envPath)
		gloRegistry = defaultRegistry(modelName)
	})
	if gloRegistry == nil {
		panic("Failed to get global registry")
	}
	return gloRegistry
}

// Load environment variables from .env file
func LoadEnvVars2Config(envPath string) {
	// Check if the .env file exists, if not, try to find in the current directory
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		if _, err := os.Stat("./.env"); err == nil {
			envPath = "./.env"
		}
	}

	// Load environment variables
	if err := godotenv.Load(envPath); err != nil {
		GetLogger().Warn("No .env file found at " + envPath + ", proceeding with existing environment variables")
	}

	// config.GEMINI_API_KEY = os.Getenv("GEMINI_API_KEY")
	// config.SILICONFLOW_API_KEY = os.Getenv("SILICONFLOW_API_KEY")
	config.DASHSCOPE_API_KEY = os.Getenv("DASHSCOPE_API_KEY")
	config.PINECONE_API_KEY = os.Getenv("PINECONE_API_KEY")
	config.COHERE_API_KEY = os.Getenv("COHERE_API_KEY")

	if config.COHERE_API_KEY == "" {
		GetLogger().Warn("COHERE_API_KEY missing in the environment variables. Please check out.")
	}
	// if config.GEMINI_API_KEY == "" {
	// 	GetLogger().Warn("GEMINI_API_KEY missing in the environment variables. Please check out.")
	// }
	// if config.SILICONFLOW_API_KEY == "" {
	// 	GetLogger().Warn("SILICONFLOW_API_KEY missing in the environment variables. Please check out.")
	// }
	if config.DASHSCOPE_API_KEY == "" {
		GetLogger().Warn("DASHSCOPE_API_KEY missing in the environment variables. Please check out.")
	}
	if config.PINECONE_API_KEY == "" {
		GetLogger().Warn("PINECONE_API_KEY missing in the environment variables. Please check out.")
	}
}

func defaultRegistry(modelName string) *genkit.Genkit {
	ctx := context.Background()
	plugins := []api.Plugin{}
	if config.SILICONFLOW_API_KEY != "" {
		plugins = append(plugins, &siliconflow.SiliconFlow{
			APIKey: config.SILICONFLOW_API_KEY,
		})
	}
	if config.GEMINI_API_KEY != "" {
		plugins = append(plugins, &googlegenai.GoogleAI{
			APIKey: config.GEMINI_API_KEY,
		})
	}
	if config.DASHSCOPE_API_KEY != "" {
		plugins = append(plugins, &dashscope.DashScope{
			APIKey: config.DASHSCOPE_API_KEY,
		})
	}
	if config.PINECONE_API_KEY != "" {
		plugins = append(plugins, &pinecone.Pinecone{
			APIKey: config.PINECONE_API_KEY,
		})
	}

	registry := genkit.Init(ctx,
		genkit.WithPlugins(plugins...),
		genkit.WithDefaultModel(modelName),
		genkit.WithPromptDir(config.PROMPT_DIR_PATH),
	)
	return registry
}

func DevLogger() *slog.Logger {
	slog.SetDefault(
		slog.New(
			prettylog.NewHandler(&slog.HandlerOptions{
				Level:       slog.LevelDebug,
				AddSource:   true,
				ReplaceAttr: nil,
			}),
		),
	)
	return slog.Default()
}

func ProductionLogger() *slog.Logger {
	slog.SetDefault(
		slog.New(
			prettylog.NewHandler(&slog.HandlerOptions{
				Level:       slog.LevelInfo,
				AddSource:   false,
				ReplaceAttr: nil,
			}),
		),
	)
	return slog.Default()
}

func GetLogger() *slog.Logger {
	if gloLogger == nil {
		gloLogger = ProductionLogger()
	}
	return gloLogger
}
