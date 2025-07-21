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
	"context"
	"fmt"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/chat"
	"github.com/apache/dubbo-kubernetes/operator/cmd/cluster"
	"github.com/chzyer/readline"
	"github.com/spf13/cobra"
	"os"

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
	llmClient, err := chat.NewOpenAIClient()
	if err != nil {
		fmt.Println("OPENAI_API_KEY is not set")
		return
	}
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

	fmt.Println(`Welcome to Dubboctl AI. I’m your command-line assistant!
  reset  Reset the conversation context
  exit   Exit the program`,
		20*time.Millisecond)

	prompt := `Introduce yourself as the command-line assistant responsible for developing Dubboctl.`
	resp, err := llmClient.Call(context.TODO(), prompt)
	if err != nil {
		fmt.Printf("call failed：%v\n", err)
		return
	}
	fmt.Println(resp)
}
