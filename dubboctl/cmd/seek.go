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
		fmt.Fprintln(os.Stderr, "Unable to initialize command line：", err)
		os.Exit(1)
	}
	defer rl.Close()

	typewriter(`Welcome to Dubboctl AI, your command-line assistant!

  reset Resets the conversation context

  exit Exits the program

Example:
  1. I want to create a Go project named dubbogo-application and use the common template library.
  2. I want to build an image named john/testapp:latest.
  3. I want to view the list of template libraries.
  4. I want to add a template library named simple and located at https://example.com.
  5. I want to delete a template library named simple.
  6. I want to deploy an image named john/testapp:latest, with the default namespace and port 8080.
  7. I want to delete the image. `, 20*time.Millisecond)

	prompt := `Introduce myself as the command line assistant responsible for the development of Dubboctl`
	resp, err := llmClient.Call(context.TODO(), prompt)
	if err != nil {
		fmt.Printf("call failed：%v\n", err)
		return
	}
	typewriter(resp, 20*time.Millisecond)

	for {
		line, err := rl.Readline()
		if err != nil {
			fmt.Println("Exit Program")
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
			fmt.Print("Confirm exit？(y/N) ")
			ans, _ := bufio.NewReader(os.Stdin).ReadString('\n')
			if strings.ToLower(strings.TrimSpace(ans)) == "y" {
				fmt.Println("goodbye！")
				return
			}

		case "reset":
			clearScreen()
			chatMemory = memory.NewConversationBuffer()
			chain = chains.NewConversation(llmClient, chatMemory)
			fmt.Println("Conversation context reset.")

		case "sdk", "image", "repo":
			if args == "" {
				fmt.Printf("Please provide content after '%s', or type help to see examples.\n", cmd)
				continue
			}
			fmt.Print("AI is thinking...")
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
			_, err := chains.Run(context.Background(), chain, input)
			if err != nil {
				fmt.Printf("AI processing errors: %v\n", err)
				continue
			}

			intent, err := classifyIntent(input, llmClient)
			if err != nil {
				fmt.Printf("I can't tell what you meant, please be more specific：%v\n", err)
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
				result = "I don't understand your request at the moment."
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
		llms.TextParts(llms.ChatMessageTypeSystem, `You are an intent recognition assistant. The user's intent can only be one of three: "sdk", "image", or "repo". Return only one of these, no more. For example:
"I want to generate a project" → sdk
"I want to build an image" → image
"I want to view the template library" → repo`),
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
	return "", fmt.Errorf("Unclear intentions: %s", intent)
}
