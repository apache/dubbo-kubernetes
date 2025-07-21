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
