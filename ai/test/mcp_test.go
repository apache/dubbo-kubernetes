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

package test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"dubbo-admin-ai/config"
	"dubbo-admin-ai/manager"
	"dubbo-admin-ai/plugins/dashscope"
	"dubbo-admin-ai/tools"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

func TestMCP(t *testing.T) {
	ctx := context.Background()
	g := manager.Registry(dashscope.Qwen_max.Key(), config.PROJECT_ROOT+"/.env", manager.DevLogger())

	mcpToolManager, err := tools.NewMCPToolManager(g, "mcpHost")
	if err != nil {
		t.Fatalf("failed to create MCP tool manager: %v", err)
	}

	toolRefs := mcpToolManager.ToolRefs()
	prompt, err := os.ReadFile(config.PROMPT_DIR_PATH + "/agentTool.txt")
	if err != nil {
		t.Fatalf("failed to read prompt file: %v", err)
	}

	resp, err := genkit.Generate(ctx, g,
		ai.WithSystem(string(prompt)),
		ai.WithPrompt("What are the existing namespaces?"),
		ai.WithTools(toolRefs...),
		ai.WithOutputType(tools.ToolOutput{}),
	)

	if err != nil {
		t.Fatalf("failed to generate text: %v", err)
	}

	manager.GetLogger().Info("Generated response:", "text", resp.Text())
}

func TestMCPFlow(t *testing.T) {
	g := manager.Registry(dashscope.Qwen3.Key(), config.PROJECT_ROOT+"/.env", manager.DevLogger())
	flow := genkit.DefineFlow(g, "mcpTest",
		func(ctx context.Context, userPrompt string) (string, error) {
			mcpToolManager, err := tools.NewMCPToolManager(g, "mcpHost")
			if err != nil {
				return "", fmt.Errorf("failed to create MCP tool manager: %v", err)
			}

			toolRefs := mcpToolManager.ToolRefs()
			prompt, err := os.ReadFile(config.PROMPT_DIR_PATH + "/agentSystem.txt")
			if err != nil {
				return "", fmt.Errorf("failed to read prompt file: %v", err)
			}

			resp, err := genkit.Generate(ctx, g,
				ai.WithSystem(string(prompt)),
				ai.WithPrompt(userPrompt),
				ai.WithTools(toolRefs...),
				ai.WithReturnToolRequests(true),
			)

			if err != nil {
				return "", fmt.Errorf("failed to generate text: %v", err)
			}

			return resp.Text(), nil
		})

	resp, err := flow.Run(context.Background(), "List all namespaces in the Kubernetes cluster")
	if err != nil {
		t.Fatalf("failed to run MCP flow: %v", err)
	}
	manager.GetLogger().Info("MCP Flow response:", "response", resp)
}
