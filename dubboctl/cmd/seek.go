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

package cmd

import (
	"bufio"
	"context"
	"fmt"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/chat"
	"github.com/apache/dubbo-kubernetes/operator/cmd/cluster"
	"github.com/chzyer/readline"
	"github.com/spf13/cobra"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms"
	llmopenai "github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/memory"
	"os"
	"strings"

	"time"
)

func SeekCmd() *cobra.Command {
	rootArgs := &cluster.RootArgs{}
	gpt := chatGPTCmd()
	s := &cobra.Command{
		Use:   "seek",
		Short: "Seek help from a large-scale AI model.",
	}
	cluster.AddFlags(s, rootArgs)
	cluster.AddFlags(gpt, rootArgs)
	s.AddCommand(gpt)
	return s
}

func chatGPTCmd() *cobra.Command {
	gpt := &cobra.Command{
		Use:   "chatgpt",
		Short: "Start an interactive ChatGPT session in your terminal",
		Args:  cobra.ExactArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			runChatGPT()
		},
	}
	return gpt
}

func runChatGPT() {
	chatMemory := memory.NewConversationBuffer()
	llmClient, err := chat.NewOpenAIClient()
	if err != nil {
		fmt.Println("OPENAI_API_KEY is not set")
		return
	}
	chain := chains.NewConversation(llmClient, chatMemory)

	rl, err := readline.NewEx(&readline.Config{
		Prompt:      "\033[1m\033[94m⌘ Dubboctl AI >\033[0m ",
		HistoryFile: "/tmp/dubboctl_ai.history",
		AutoComplete: readline.NewPrefixCompleter(
			readline.PcItem("sdk"),
			readline.PcItem("image"),
			readline.PcItem("repo"),
			readline.PcItem("exit"),
		),
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "无法初始化命令行：", err)
		os.Exit(1)
	}
	defer rl.Close()

	typewriter(`欢迎使用 Dubboctl AI，我是你的命令行助手！
  reset 重置对话上下文
  exit 退出程序

示例：
  1. 我要创建一个 go 的项目，项目名称是 dubbogo-application，模板库是 common。
  2. 我要构建镜像，镜像信息是 john/testapp:latest。
  3. 我想看看一下模板库列表。
  4. 我想添加模板库，模板名称是 simple，模板地址是 https://example.com。
  5. 我想删除模板库，模版名称是 simple。
  6. 我要部署镜像，镜像名称是 john/testapp:latest，命名空间是 default，端口是 8080。
  7. 我要删除镜像。`, 20*time.Millisecond)

	prompt := `介绍自己负责 Dubboctl 的开发命令行助手`
	resp, err := llmClient.Call(context.TODO(), prompt)
	if err != nil {
		fmt.Printf("调用失败：%v\n", err)
		return
	}
	typewriter(resp, 20*time.Millisecond)

	for {
		line, err := rl.Readline()
		if err != nil {
			fmt.Println("退出程序")
			break
		}
		input := strings.TrimSpace(line)
		if input == "" {
			continue
		}

		parts := strings.Fields(input)
		cmd := parts[0]
		var args string
		if len(parts) > 1 {
			args = strings.Join(parts[1:], " ")
		}

		switch cmd {
		case "exit":
			fmt.Print("确认退出？(y/N) ")
			ans, _ := bufio.NewReader(os.Stdin).ReadString('\n')
			if strings.ToLower(strings.TrimSpace(ans)) == "y" {
				fmt.Println("再见！")
				return
			}

		case "reset":
			clearScreen()
			chatMemory = memory.NewConversationBuffer()
			chain = chains.NewConversation(llmClient, chatMemory)
			fmt.Println("已重置对话上下文。")

		case "sdk", "image", "repo":
			if args == "" {
				fmt.Printf("请在 '%s' 后提供内容，输入 help 查看示例。\n", cmd)
				continue
			}
			fmt.Print("AI 正在思考...")
			time.Sleep(200 * time.Millisecond)

			var result string
			switch cmd {
			case "sdk":
				result = chatGPTFromSdk(args)
			case "image":
				result = chatGPTFromImage(args)
			case "repo":
				result = chatGPTFromRepo(args)
			}

			fmt.Println()
			typewriter(result, 20*time.Millisecond)
			continue

		default:
			// 非命令，尝试自然语言处理
			_, err := chains.Run(context.Background(), chain, input)
			if err != nil {
				fmt.Printf("AI 处理出错: %v\n", err)
				continue
			}

			intent, err := classifyIntent(input, llmClient)
			if err != nil {
				fmt.Printf("无法判断你的意图，请更具体一点：%v\n", err)
				continue
			}

			var result string
			switch intent {
			case "sdk":
				result = chatGPTFromSdk(input)
			case "image":
				result = chatGPTFromImage(input)
			case "repo":
				result = chatGPTFromRepo(input)
			default:
				result = "我暂时无法理解你的请求。"
			}

			typewriter(result, 20*time.Millisecond)
		}
	}
}

func clearScreen() {
	fmt.Print("\033[H\033[2J")
}

func typewriter(text string, delay time.Duration) {
	for _, r := range text {
		fmt.Print(string(r))
		time.Sleep(delay)
	}
	fmt.Println()
}

func chatGPTFromSdk(input string) string {
	client, err := chat.NewOpenAIClient()
	if err != nil {
		return err.Error()
	}
	resp := chat.SdkFunctionCall(input, client)
	return resp
}

func chatGPTFromImage(input string) string {
	client, err := chat.NewOpenAIClient()
	if err != nil {
		return err.Error()
	}
	resp := chat.ImageFunctionCall(input, client)
	return resp
}

func chatGPTFromRepo(input string) string {
	client, err := chat.NewOpenAIClient()
	if err != nil {
		return err.Error()
	}
	resp := chat.RepoFunctionCall(input, client)
	return resp
}

func classifyIntent(input string, client *llmopenai.LLM) (string, error) {
	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, `你是一个意图识别助手。用户的意图只能是三种之一："sdk", "image", "repo"。只返回其中一个，不要多说。例如：
"我想生成一个项目" → sdk
"我想构建镜像" → image
"我想看模板库" → repo`),
		llms.TextParts(llms.ChatMessageTypeHuman, input),
	}
	resp, err := client.GenerateContent(context.TODO(), messages)
	if err != nil {
		return "", err
	}
	intent := strings.TrimSpace(resp.Choices[0].Content)
	if intent == "sdk" || intent == "image" || intent == "repo" {
		return intent, nil
	}
	return "", fmt.Errorf("意图不明确: %s", intent)
}
