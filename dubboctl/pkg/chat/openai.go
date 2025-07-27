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

package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
	"github.com/tmc/langchaingo/llms"
	llmopenai "github.com/tmc/langchaingo/llms/openai"
	"os/exec"
)

func NewOpenAIClient() (*llmopenai.LLM, error) {
	llmClient, err := llmopenai.New()
	if err != nil {
		return nil, err
	}
	return llmClient, nil
}

func SdkFunctionCall(input string, client *llmopenai.LLM) string {
	f := llms.FunctionDefinition{
		Name:        "sdkResource",
		Description: "The sdk subcommand generates an sdk sample provided by Dubbo supported languages.",
		Parameters: jsonschema.Definition{
			Type: jsonschema.Object,
			Properties: map[string]jsonschema.Definition{
				"language": {
					Type:        jsonschema.String,
					Description: "根据用户输入指定的语言，例如 go 或 java",
				},
				"template": {
					Type:        jsonschema.String,
					Description: "根据用户输入用户指定的模板库，例如 common",
				},
				"dirname": {
					Type:        jsonschema.String,
					Description: "根据用户输入指定的项目名称，例如 mydubbo",
				},
			},
			Required: []string{"language", "template", "dirname"},
		},
	}

	t := []llms.Tool{
		{
			Type:     string(openai.ToolTypeFunction),
			Function: &f,
		},
	}

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, `
你是一个 Dubbo SDK 项目创建助手。当用户请求你创建或生成时，请帮助补全以下参数：
请根据以下规则进行补全：

1. 如果用户只提供语言（如 "go"），你可以：
   - 默认使用 template 为 "common"
   - 默认目录名为 "app"

2. 如果用户提供语言和目录名，但没有指定 template：
   - 使用 template 为 "common"

3. 如果用户什么都没说（比如只说“帮我创建或生成相关的关键字”），你可以默认：
   - language: "go"
   - template: "common"
   - dirname: 你随机生成项目名称。

无论使用哪种默认值，请在回答中清楚地告诉用户你用了哪些默认值。
并引导用户可以通过命令 dubboctl repo list 查看模板库。

注意：如果用户不是在创建/生成的上下文中发问（比如问部署、镜像等），你不要回复。
`),
		llms.TextParts(llms.ChatMessageTypeHuman, input),
	}

	resp, err := client.GenerateContent(context.TODO(), messages, llms.WithTools(t))

	msg := resp.Choices[0].ToolCalls
	if len(msg) != 1 {
		return "我没有完全理解你的意思，您是想创建 go 还是 java 的 sdk 项目？"
	}

	fmt.Printf("OpenAI 请求函数 %s，参数为：%s\n", msg[0].FunctionCall.Name, msg[0].FunctionCall.Arguments)

	res, err := call(client, msg[0].FunctionCall.Name, msg[0].FunctionCall.Arguments)
	if err != nil {
		return err.Error()
	}
	return res
}

func ImageFunctionCall(input string, client *llmopenai.LLM) string {
	f1 := llms.FunctionDefinition{
		Name:        "imageHubResource",
		Description: "The hub subcommand used to build and push images",
		Parameters: jsonschema.Definition{
			Type: jsonschema.Object,
			Properties: map[string]jsonschema.Definition{
				"image_info": {
					Type:        jsonschema.String,
					Description: "用户输入的镜像信息，例如 john/testapp:latest",
				},
			},
			Required: []string{"image_info"},
		},
	}
	f2 := llms.FunctionDefinition{
		Name:        "imageDeployResource",
		Description: "The deploy subcommand used to deploy to cluster",
		Parameters: jsonschema.Definition{
			Type: jsonschema.Object,
			Properties: map[string]jsonschema.Definition{
				"deploy_type": {
					Type:        jsonschema.String,
					Description: "判断用户部署操作，例如 create 或 --delete",
				},
				"image_info": {
					Type:        jsonschema.String,
					Description: "用户输入的部署镜像信息，例如 john/testapp:latest",
				},
				"namespace": {
					Type:        jsonschema.String,
					Description: "用户输入的部署命名空间，例如 default",
				},
				"port": {
					Type:        jsonschema.String,
					Description: "用户输入的部署端口，例如 80",
				},
			},
			Required: []string{"deploy_type", "image_info", "namespace", "port"},
		},
	}

	t := []llms.Tool{
		{Type: string(openai.ToolTypeFunction), Function: &f1},
		{Type: string(openai.ToolTypeFunction), Function: &f2},
	}

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, `
你是 Dubbo SDK 项目的镜像助手，可以帮助用户执行以下操作：

1. 构建或推送镜像（使用 imageHubResource）
2. 部署或删除镜像（使用 imageDeployResource）

根据用户的输入判断调用哪个函数：
- 如果用户提到“构建”、“打包”、“推送”，调用 imageHubResource，参数为 image_info。
- 如果用户提到“部署”、“上线”、“运行”，调用 imageDeployResource，参数包括：
  - deploy_type：部署类型，create 表示部署
  - image_info：镜像名称；
  - namespace：部署的命名空间。
  - port 部署的端口。
- 如果用户提到“删除”，delete 表示删除；调用 imageDeployResource，直接删除。

注意：
- 如果用户没有提供 deploy_type。
- 如果用户没有提供 namespace，默认使用 "default"。
- 如果用户没有提供 port，默认使用 80。
- 如果用户没说清楚 intent，请提问澄清。
- 请友好引导用户补充缺失的信息。
`),
		llms.TextParts(llms.ChatMessageTypeHuman, input),
	}

	resp, err := client.GenerateContent(context.TODO(), messages, llms.WithTools(t))

	msg := resp.Choices[0].ToolCalls

	if len(msg) != 1 {
		if resp.Choices[0].Content != "" {
			return resp.Choices[0].Content
		}
		return "我没有完全理解你的意思，您是想构建或推送镜像还是部署或删除镜像？"
	}

	fmt.Printf("OpenAI 请求函数 %s，参数为：%s\n", msg[0].FunctionCall.Name, msg[0].FunctionCall.Arguments)

	res, err := call(client, msg[0].FunctionCall.Name, msg[0].FunctionCall.Arguments)
	if err != nil {
		return err.Error()
	}
	return res
}

func RepoFunctionCall(input string, client *llmopenai.LLM) string {
	f := llms.FunctionDefinition{
		Name:        "repoResource",
		Description: "The repo command Manage existing Dubbo SDK module libraries",
		Parameters: jsonschema.Definition{
			Type: jsonschema.Object,
			Properties: map[string]jsonschema.Definition{
				"name": {
					Type:        jsonschema.String,
					Description: "用户输入的模板库名称操作",
				},
				"url": {
					Type:        jsonschema.String,
					Description: "用户输入的模板库地址操作",
				},
				"repo_type": {
					Type:        jsonschema.String,
					Description: "用户输入的模板库操作",
				},
			},
			Required: []string{"repo_type"},
		},
	}

	t := []llms.Tool{
		{
			Type:     string(openai.ToolTypeFunction),
			Function: &f,
		},
	}

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, "你是一个帮助用户模板库管理的助手，例如：dubboctl repo add/list/remove 的操作"),
		llms.TextParts(llms.ChatMessageTypeHuman, input),
	}

	resp, err := client.GenerateContent(context.TODO(), messages, llms.WithTools(t))

	msg := resp.Choices[0].ToolCalls
	if len(msg) != 1 {
		if resp.Choices[0].Content != "" {
			return resp.Choices[0].Content
		}
		return "我没有完全理解你的意思，您是想添加、查看还是删除模板库？"
	}

	fmt.Printf("OpenAI 请求函数 %s，参数为：%s\n", msg[0].FunctionCall.Name, msg[0].FunctionCall.Arguments)

	res, err := call(client, msg[0].FunctionCall.Name, msg[0].FunctionCall.Arguments)
	if err != nil {
		return err.Error()
	}
	return res
}

func call(client *llmopenai.LLM, name, argument string) (string, error) {
	if name == "sdkResource" {
		sdkParams := struct {
			Language string `json:"language"`
			Template string `json:"template"`
			Dirname  string `json:"dirname"`
		}{}
		err := json.Unmarshal([]byte(argument), &sdkParams)
		if err != nil {
			return "", err
		}
		// 参数不完整，调用大模型生成默认值
		if sdkParams.Language == "" || sdkParams.Template == "" || sdkParams.Dirname == "" {
			prompt := fmt.Sprintf(
				"用户想创建一个 Dubbo SDK 项目，但参数不全。请你用合适的默认值填补缺失字段，已有参数：language=%s, template=%s, dirname=%s。请返回完整 JSON。",
				sdkParams.Language, sdkParams.Template, sdkParams.Dirname,
			)

			resp, err := client.Call(context.TODO(), prompt)
			if err != nil {
				return "", fmt.Errorf("大模型补全失败: %v", err)
			}

			err = json.Unmarshal([]byte(resp), &sdkParams)
			if err != nil {
				return "", fmt.Errorf("大模型生成的 JSON 无效: %v", err)
			}
		}

		return sdkResource(sdkParams.Language, sdkParams.Template, sdkParams.Dirname)
	}
	if name == "imageHubResource" {
		imageParams := struct {
			ImageInfo string `json:"image_info"`
		}{}
		err := json.Unmarshal([]byte(argument), &imageParams)
		if err != nil {
			return "", err
		}
		if imageParams.ImageInfo == "" {
			prompt := fmt.Sprintf(
				"用户想将 Dubbo SDK 项目构建成容器镜像，但参数需要交互模式，如果用户不填信息，就会发生错误，例如：Error: failed to build the application: invalid image name '': image is a required parameter。已有参数：imageInfo=%s。请返回完整 JSON。",
				imageParams.ImageInfo,
			)

			resp, err := client.Call(context.TODO(), prompt)
			if err != nil {
				return "", fmt.Errorf("大模型补全失败: %v", err)
			}

			err = json.Unmarshal([]byte(resp), &imageParams)
			if err != nil {
				return "", fmt.Errorf("大模型生成的 JSON 无效: %v", err)
			}
		}

		return imageHubResource(imageParams.ImageInfo)
	}
	if name == "imageDeployResource" {
		imageParams := struct {
			ImageInfo  string `json:"image_info"`
			Namespace  string `json:"namespace"`
			Port       string `json:"port"`
			DeployType string `json:"deploy_type"`
		}{}
		err := json.Unmarshal([]byte(argument), &imageParams)
		if err != nil {
			return "", err
		}
		if imageParams.DeployType == "" || (imageParams.DeployType == "create" && (imageParams.Port == "" || imageParams.Namespace == "" || imageParams.ImageInfo == "")) || (imageParams.DeployType == "delete") {
			prompt := fmt.Sprintf("用户想部署镜像到集群里，create 操作补全以下字段：port=%s namespace=%s, imageInfo=%s； delete 操作直接删除，并返回 JSON 格式。",
				imageParams.Port, imageParams.Namespace, imageParams.ImageInfo)
			resp, err := client.Call(context.TODO(), prompt)
			if err != nil {
				return "", fmt.Errorf("大模型补全失败: %v", err)
			}
			err = json.Unmarshal([]byte(resp), &imageParams)
			if err != nil {
				return "", fmt.Errorf("大模型返回的 JSON 无效: %v", err)
			}
		}
		return imageDeployResource(imageParams.DeployType, imageParams.Port, imageParams.Namespace, imageParams.ImageInfo)
	}
	if name == "repoResource" {
		Params := struct {
			Name     string `json:"name"`
			Url      string `json:"url"`
			RepoType string `json:"repo_type"`
		}{}
		err := json.Unmarshal([]byte(argument), &Params)
		if err != nil {
			return "", fmt.Errorf("参数解析失败: %v", err)
		}
		if Params.RepoType == "" || (Params.RepoType == "add" && (Params.Name == "" || Params.Url == "")) || (Params.RepoType == "remove" && Params.Name == "") {
			prompt := fmt.Sprintf("用户想进行模板库操作，add 操作补全以下字段：name=%s, url=%s list 操作直接输出，remove 补全以下字段：name=%s，并返回 JSON 格式。",
				Params.Name, Params.Url, Params.RepoType)
			resp, err := client.Call(context.TODO(), prompt)
			if err != nil {
				return "", fmt.Errorf("大模型补全失败: %v", err)
			}
			err = json.Unmarshal([]byte(resp), &Params)
			if err != nil {
				return "", fmt.Errorf("大模型返回的 JSON 无效: %v", err)
			}
		}
		return repoResource(Params.RepoType, Params.Name, Params.Url)
	}
	return "", nil
}

func sdkResource(language, template, dirname string) (string, error) {
	_, err := exec.LookPath("dubboctl")
	if err != nil {
		return "", errors.New("未找到 dubboctl，请确保已安装并配置在 PATH 中")
	}
	// 构建命令
	cmd := exec.Command("dubboctl", "create", "sdk",
		"--language", language,
		"--template", template,
		"--dirname", dirname,
	)
	// 获取命令输出
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("命令执行失败：%s\n错误：%v", string(output), err)
	}
	fmt.Printf("命令执行成功，输出如下：\n%s", string(output))

	var client *llmopenai.LLM
	client, err = NewOpenAIClient()
	if err != nil {
		return "", fmt.Errorf("创建 LLM 客户端失败: %v", err)
	}
	prompt := fmt.Sprintf(`我刚刚帮用户执行了 dubboctl 命令，以下是输出：%s 请用自然语言告诉用户项目已经创建，必要时也可以提醒他们下一步操作，例如可以执行 dubboctl repo list 查看模板列表等。`, string(output))

	resp, err := client.Call(context.TODO(), prompt)
	if err != nil {
		return "", fmt.Errorf("生成自然语言描述失败: %v", err)
	}
	return resp, nil
}

func imageHubResource(imageInfo string) (string, error) {
	client, err := NewOpenAIClient()
	if err != nil {
		return "", fmt.Errorf("创建 LLM 客户端失败: %v", err)
	}
	_, err = exec.LookPath("dubboctl")
	if err != nil {
		return "", errors.New("未找到 dubboctl，请确保已安装并配置在 PATH 中")
	}

	cmd := exec.Command("dubboctl", "image", "hub", "-b",
		"--imageInfo", imageInfo,
	)

	prompt := fmt.Sprintf(`根据用户输入的中文或英文进行回答，发生错误根据举例告诉用户是不是没有创建或切换项目目录即可，例如：Error: does not contain an initialized sdk
	注意：我不要你任何语气词。`)

	resp, err := client.Call(context.TODO(), prompt)
	if err != nil {
		return "", fmt.Errorf("生成自然语言描述失败: %v", err)
	}
	fmt.Println(resp)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("命令执行失败：%s\n错误：%v", string(output), err)
	}
	fmt.Printf("命令执行成功，输出如下：\n%s", string(output))

	prompt = fmt.Sprintf(`我刚刚帮用户执行了 dubboctl 命令，总结一下做了什么，注意：我不要你任何语气词。以下是输出：%s`, string(output))
	resp, err = client.Call(context.TODO(), prompt)
	if err != nil {
		return "", fmt.Errorf("生成自然语言描述失败: %v", err)
	}
	return resp, nil
}

func imageDeployResource(deployType, port, namespace, imageInfo string) (string, error) {
	client, err := NewOpenAIClient()
	if err != nil {
		return "", fmt.Errorf("创建 LLM 客户端失败: %v", err)
	}
	_, err = exec.LookPath("dubboctl")
	if err != nil {
		return "", errors.New("未找到 dubboctl，请确保已安装并配置在 PATH 中")
	}

	var output string
	switch deployType {
	case "create":
		output, err = resourceCreate(port, namespace, imageInfo)
		if err != nil {
			return "", err
		}
	case "--delete":
		output, err = resourceDelete(deployType)
		if err != nil {
			return "", err
		}
	}

	prompt := fmt.Sprintf(`我刚刚帮用户执行了 dubboctl 命令，总结一下做了什么，注意：我不要你任何语气词。以下是输出：%s`, string(output))
	resp, err := client.Call(context.TODO(), prompt)
	if err != nil {
		return "", fmt.Errorf("生成自然语言描述失败: %v", err)
	}
	return resp, nil
}

func repoResource(repoType, name, url string) (string, error) {
	client, err := NewOpenAIClient()
	if err != nil {
		return "", fmt.Errorf("创建 LLM 客户端失败: %v", err)
	}
	_, err = exec.LookPath("dubboctl")
	if err != nil {
		return "", errors.New("未找到 dubboctl，请确保已安装并配置在 PATH 中")
	}

	var output string
	switch repoType {
	case "add":
		if name == "" || url == "" {
			return "", fmt.Errorf("add 操作需要提供 name 和 url")
		}
		output, err = repoAdd(repoType, name, url)
		if err != nil {
			return "", err
		}
	case "list":
		output, err = repoList(repoType)
		if err != nil {
			return "", err
		}
	case "remove":
		if name == "" {
			return "", fmt.Errorf("remove 操作需要提供 name")
		}
		output, err = repoRemove(repoType, name)
		if err != nil {
			return "", err
		}
	default:
		return "", fmt.Errorf("未知操作类型: %s", repoType)
	}

	prompt := fmt.Sprintf("我刚刚帮用户执行了 dubboctl repo %s 操作，总结一下做了什么，注意：我不要你任何语气词。以下是输出：%s", repoType, output)
	resp, err := client.Call(context.TODO(), prompt)
	if err != nil {
		return "", fmt.Errorf("生成自然语言描述失败: %v", err)
	}
	return resp, nil
}

func repoAdd(repoType, name, url string) (string, error) {
	cmd := exec.Command("dubboctl", "repo", repoType, name, url)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("命令执行失败：%s\n错误：%v", string(output), err)
	}
	fmt.Printf("命令执行成功，输出如下：\n%s", string(output))
	return string(output), nil
}

func repoList(repoType string) (string, error) {
	cmd := exec.Command("dubboctl", "repo", repoType)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("命令执行失败：%s\n错误：%v", string(output), err)
	}
	fmt.Printf("命令执行成功，输出如下：\n%s", string(output))
	return string(output), nil
}

func repoRemove(repoType, name string) (string, error) {
	cmd := exec.Command("dubboctl", "repo", repoType, name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("命令执行失败：%s\n错误：%v", string(output), err)
	}
	fmt.Printf("命令执行成功，输出如下：\n%s", string(output))
	return string(output), nil
}

func resourceCreate(port, namespace, imageInfo string) (string, error) {
	cmd := exec.Command("dubboctl", "image", "deploy",
		"--namespace", namespace,
		"--imageInfo", imageInfo,
		"--port", port)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("命令执行失败：%s\n错误：%v", string(output), err)
	}
	fmt.Printf("命令执行成功，输出如下：\n%s", string(output))
	return string(output), nil
}

func resourceDelete(deployType string) (string, error) {
	cmd := exec.Command("dubboctl", "image", "deploy", deployType)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("命令执行失败：%s\n错误：%v", string(output), err)
	}
	fmt.Printf("命令执行成功，输出如下：\n%s", string(output))
	return string(output), nil
}
