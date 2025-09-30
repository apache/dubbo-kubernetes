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

package tools

import (
	"context"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/mcp"
)

type MCPToolManager struct {
	registry       *genkit.Genkit
	mcpHost        *mcp.MCPHost
	availableTools map[string]ai.Tool
}

func DefineMCPHost(g *genkit.Genkit, hostName string, mcpNameCmdMap map[string][]string) (*mcp.MCPHost, error) {
	servers := make([]mcp.MCPServerConfig, 0, len(mcpNameCmdMap))
	for key, value := range mcpNameCmdMap {
		server := mcp.MCPServerConfig{
			Name: key,
			Config: mcp.MCPClientOptions{
				Name: key,
				Stdio: &mcp.StdioConfig{
					Command: value[0],
					Args:    value[1:],
				},
			},
		}
		servers = append(servers, server)
	}
	host, err := mcp.NewMCPHost(g, mcp.MCPHostOptions{Name: hostName, MCPServers: servers})
	if err != nil {
		return nil, err
	}
	return host, nil
}

func NewMCPToolManager(g *genkit.Genkit, hostName string) (*MCPToolManager, error) {
	mcps := map[string][]string{
		"kubernetes": {
			"npx",
			"-y",
			"kubernetes-mcp-server@latest",
		},
		// "prometheus": {
		// 	"docker",
		// 	"run",
		// 	"-i",
		// 	"--rm",
		// 	"-e",
		// 	config.PROMETHEUS_URL,
		// 	"ghcr.io/pab1it0/prometheus-mcp-server:latest",
		// },
	}

	host, err := DefineMCPHost(g, hostName, mcps)
	if err != nil {
		return nil, err
	}

	activeTools, err := host.GetActiveTools(context.Background(), g)
	if err != nil {
		return nil, err
	}

	var availableTools = make(map[string]ai.Tool, len(activeTools))
	for _, tool := range activeTools {
		availableTools[tool.Name()] = tool
	}

	return &MCPToolManager{
		registry:       g,
		mcpHost:        host,
		availableTools: availableTools,
	}, nil
}

func (mtm *MCPToolManager) AllTools() []ai.Tool {
	var tools []ai.Tool
	for _, tool := range mtm.availableTools {
		tools = append(tools, tool)
	}
	return tools
}

func (mtm *MCPToolManager) ToolRefs() (toolRef []ai.ToolRef) {
	for _, tool := range mtm.availableTools {
		toolRef = append(toolRef, tool)
	}
	return toolRef
}

func (mtm *MCPToolManager) GetToolByName(name string) ai.Tool {
	return mtm.availableTools[name]
}
