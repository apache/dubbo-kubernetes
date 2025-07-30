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
					Description: "A language specified by user input, such as go or java",
				},
				"template": {
					Type:        jsonschema.String,
					Description: "Based on user input, a user-specified template library, such as common",
				},
				"dirname": {
					Type:        jsonschema.String,
					Description: "The project name specified by the user, such as mydubbo",
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
You are a Dubbo SDK project creation assistant. When a user requests you to create or generate a project, please help complete the following parameters:
Please complete according to the following rules:

1. If the user only provides a language (such as "go"), you can:
- Default template to "common"
- Default directory name to "app"

2. If the user provides a language and directory name but does not specify a template:
- Default template to "common"

3. If the user does not specify anything (for example, only "Help me create or generate related keywords"), you can default to:
- language: "go"
- template: "common"
- dirname: You randomly generate a project name.

Regardless of which default values you use, please clearly explain the defaults in your response.

And guide the user to view the template repository using the command dubboctl repo list.

Note: If the user's question is not in the context of creating or generating a project (such as regarding deployment or images), do not respond.
`),
		llms.TextParts(llms.ChatMessageTypeHuman, input),
	}

	resp, err := client.GenerateContent(context.TODO(), messages, llms.WithTools(t))

	msg := resp.Choices[0].ToolCalls
	if len(msg) != 1 {
		return "I don't fully understand what you mean. Do you want to create a Go or Java SDK project?"
	}

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
					Description: "Mirror information entered by the user, such as john/testapp:latest",
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
					Description: "Judge user deployment operations, such as create or --delete",
				},
				"image_info": {
					Type:        jsonschema.String,
					Description: "Deployment image information entered by the user, such as john/testapp:latest",
				},
				"namespace": {
					Type:        jsonschema.String,
					Description: "The deployment namespace entered by the user, such as default",
				},
				"port": {
					Type:        jsonschema.String,
					Description: "Deployment port entered by the user, for example 80",
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
You are an image assistant for the Dubbo SDK project, and can help users perform the following operations:

1. Build or push an image (using imageHubResource)

2. Deploy or delete an image (using imageDeployResource)

Determine which function to call based on user input:
- If the user mentions "build," "package," or "push," call imageHubResource with the image_info parameter.
- If the user mentions "deploy," "launch," or "run," call imageDeployResource with the following parameters:
- deploy_type: Deployment type, create for deployment
- image_info: Image name
- namespace: Deployment namespace
- port: Deployment port.
- If the user mentions "delete," delete for deletion; call imageDeployResource to directly delete.

Note:
- If the user does not provide a deploy_type.
- If the user does not provide a namespace, "default" is used by default.
- If the user does not provide a port, port 80 is used by default.
- If the user's intent is unclear, please ask for clarification.
- Please kindly guide users to fill in the missing information.
`),
		llms.TextParts(llms.ChatMessageTypeHuman, input),
	}

	resp, err := client.GenerateContent(context.TODO(), messages, llms.WithTools(t))

	msg := resp.Choices[0].ToolCalls

	if len(msg) != 1 {
		if resp.Choices[0].Content != "" {
			return resp.Choices[0].Content
		}
		return "I don't fully understand what you mean, do you want to build or push an image or deploy or delete an image?"
	}

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
					Description: "User-entered template library name operation",
				},
				"url": {
					Type:        jsonschema.String,
					Description: "User-entered template library address operation",
				},
				"repo_type": {
					Type:        jsonschema.String,
					Description: "Template library operations for user input",
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
		llms.TextParts(llms.ChatMessageTypeSystem, "You are an assistant who helps users manage template libraries, such as dubboctl repo add/list/remove operations"),
		llms.TextParts(llms.ChatMessageTypeHuman, input),
	}

	resp, err := client.GenerateContent(context.TODO(), messages, llms.WithTools(t))

	msg := resp.Choices[0].ToolCalls
	if len(msg) != 1 {
		if resp.Choices[0].Content != "" {
			return resp.Choices[0].Content
		}
		return "I don't fully understand what you mean. Do you want to add, view, or delete a template library?"
	}

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
		if sdkParams.Language == "" || sdkParams.Template == "" || sdkParams.Dirname == "" {
			prompt := fmt.Sprintf(
				"The user wants to create a Dubbo SDK project, but the parameters are incomplete. Please fill in the missing fields with appropriate default values. Existing parameters: language=%s, template=%s, dirname=%s. Please return a complete JSON.",
				sdkParams.Language, sdkParams.Template, sdkParams.Dirname,
			)

			resp, err := client.Call(context.TODO(), prompt)
			if err != nil {
				return "", fmt.Errorf("large model completion failed: %v", err)
			}

			err = json.Unmarshal([]byte(resp), &sdkParams)
			if err != nil {
				return "", fmt.Errorf("invalid JSON generated by large model: %v", err)
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
				"The user wants to build the Dubbo SDK project into a container image, but the parameters require interactive mode. If the user does not fill in the information, an error will occur, for example: Error: failed to build the application: invalid image name '': image is a required parameter. Existing parameter: imageInfo=%s. Please return the complete JSON.", imageParams.ImageInfo,
			)

			resp, err := client.Call(context.TODO(), prompt)
			if err != nil {
				return "", fmt.Errorf("large model completion failed: %v", err)
			}

			err = json.Unmarshal([]byte(resp), &imageParams)
			if err != nil {
				return "", fmt.Errorf("invalid JSON generated by large model: %v", err)
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
			prompt := fmt.Sprintf("If a user wants to deploy an image to a cluster, the create operation should complete the following fields: port=%s namespace=%s, imageInfo=%s. The delete operation will directly delete the image and return a JSON format.",
				imageParams.Port, imageParams.Namespace, imageParams.ImageInfo)
			resp, err := client.Call(context.TODO(), prompt)
			if err != nil {
				return "", fmt.Errorf("large model completion failed: %v", err)
			}
			err = json.Unmarshal([]byte(resp), &imageParams)
			if err != nil {
				return "", fmt.Errorf("invalid JSON generated by large model: %v", err)
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
			return "", fmt.Errorf("Parameter parsing failed: %v", err)
		}
		if Params.RepoType == "" || (Params.RepoType == "add" && (Params.Name == "" || Params.Url == "")) || (Params.RepoType == "remove" && Params.Name == "") {
			prompt := fmt.Sprintf("The user wants to operate the template library. The add operation completes the following fields: name=%s, url=%s. The list operation directly outputs, and the remove operation completes the following fields: name=%s and returns JSON format.",
				Params.Name, Params.Url, Params.RepoType)
			resp, err := client.Call(context.TODO(), prompt)
			if err != nil {
				return "", fmt.Errorf("large model completion failed: %v", err)
			}
			err = json.Unmarshal([]byte(resp), &Params)
			if err != nil {
				return "", fmt.Errorf("invalid JSON generated by large model: %v", err)
			}
		}
		return repoResource(Params.RepoType, Params.Name, Params.Url)
	}
	return "", nil
}

func sdkResource(language, template, dirname string) (string, error) {
	_, err := exec.LookPath("dubboctl")
	if err != nil {
		return "", errors.New("dubboctl not found, please make sure it is installed and configured in your PATH")
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
		return "", fmt.Errorf("Command execution failed: %s\nError: %v", string(output), err)
	}
	fmt.Printf("The command was executed successfully, and the output is as follows:\n%s", string(output))

	var client *llmopenai.LLM
	client, err = NewOpenAIClient()
	if err != nil {
		return "", fmt.Errorf("Failed to create LLM client: %v", err)
	}
	prompt := fmt.Sprintf(`I just ran the dubboctl command for the user. Here is the output: %s Please use natural language to tell the user that the project has been created. If necessary, you can also remind them of the next steps. For example, you can run dubboctl repo list to view the template list. `, string(output))

	resp, err := client.Call(context.TODO(), prompt)
	if err != nil {
		return "", fmt.Errorf("failed to generate natural language description: %v", err)
	}
	return resp, nil
}

func imageHubResource(imageInfo string) (string, error) {
	client, err := NewOpenAIClient()
	if err != nil {
		return "", fmt.Errorf("failed to create LLM client: %v", err)
	}
	_, err = exec.LookPath("dubboctl")
	if err != nil {
		return "", errors.New("dubboctl not found, please make sure it is installed and configured in your PATH")
	}

	cmd := exec.Command("dubboctl", "image", "hub", "-b",
		"--imageInfo", imageInfo,
	)

	prompt := fmt.Sprintf(`Answer based on the Chinese or English input from the user. If an error occurs, provide an example to indicate whether the project directory was not created or switched, e.g., "Error: does not contain an initialized sdk."
Note: I don't want any interjections.`)

	resp, err := client.Call(context.TODO(), prompt)
	if err != nil {
		return "", fmt.Errorf("failed to generate natural language description: %v", err)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("Command execution failed: %s\nError: %v", string(output), err)
	}
	fmt.Printf("The command was executed successfully, and the output is as follows:\n%s", string(output))

	prompt = fmt.Sprintf(`I just ran the dubboctl command for a user. Here's a summary of what was done. Note: I don't want any interjections. Here's the output: %s`, string(output))
	resp, err = client.Call(context.TODO(), prompt)
	if err != nil {
		return "", fmt.Errorf("Failed to generate natural language description: %v", err)
	}
	return resp, nil
}

func imageDeployResource(deployType, port, namespace, imageInfo string) (string, error) {
	client, err := NewOpenAIClient()
	if err != nil {
		return "", fmt.Errorf("Failed to create LLM client: %v", err)
	}
	_, err = exec.LookPath("dubboctl")
	if err != nil {
		return "", errors.New("dubboctl not found, please make sure it is installed and configured in your PATH")
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

	prompt := fmt.Sprintf(`I just ran the dubboctl command for a user. Here's a summary of what was done. Note: I don't want any interjections. Here's the output: %s`, string(output))
	resp, err := client.Call(context.TODO(), prompt)
	if err != nil {
		return "", fmt.Errorf("failed to generate natural language description: %v", err)
	}
	return resp, nil
}

func repoResource(repoType, name, url string) (string, error) {
	client, err := NewOpenAIClient()
	if err != nil {
		return "", fmt.Errorf("failed to create LLM client: %v", err)
	}
	_, err = exec.LookPath("dubboctl")
	if err != nil {
		return "", errors.New("dubboctl not found, please make sure it is installed and configured in your PATH")
	}

	var output string
	switch repoType {
	case "add":
		if name == "" || url == "" {
			return "", fmt.Errorf("The add operation requires a name and a url")
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
			return "", fmt.Errorf("The remove operation requires a name")
		}
		output, err = repoRemove(repoType, name)
		if err != nil {
			return "", err
		}
	default:
		return "", fmt.Errorf("unknown operation type: %s", repoType)
	}

	prompt := fmt.Sprintf("I just ran the dubboctl repo %s operation for a user. Here's a summary of what I did. Note: I don't want any interjections. Here's the output: %s", repoType, output)
	resp, err := client.Call(context.TODO(), prompt)
	if err != nil {
		return "", fmt.Errorf("failed to generate natural language description: %v", err)
	}
	return resp, nil
}

func repoAdd(repoType, name, url string) (string, error) {
	cmd := exec.Command("dubboctl", "repo", repoType, name, url)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("Command execution failed: %s\nError: %v", string(output), err)
	}
	fmt.Printf("The command was executed successfully, and the output is as follows:\n%s", string(output))
	return string(output), nil
}

func repoList(repoType string) (string, error) {
	cmd := exec.Command("dubboctl", "repo", repoType)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("Command execution failed: %s\nError: %v", string(output), err)
	}
	fmt.Printf("The command was executed successfully, and the output is as follows:\n%s", string(output))
	return string(output), nil
}

func repoRemove(repoType, name string) (string, error) {
	cmd := exec.Command("dubboctl", "repo", repoType, name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("Command execution failed: %s\nError: %v", string(output), err)
	}
	fmt.Printf("The command was executed successfully, and the output is as follows:\n%s", string(output))
	return string(output), nil
}

func resourceCreate(port, namespace, imageInfo string) (string, error) {
	cmd := exec.Command("dubboctl", "image", "deploy",
		"--namespace", namespace,
		"--imageInfo", imageInfo,
		"--port", port)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("Command execution failed: %s\nError: %v", string(output), err)
	}
	fmt.Printf("The command was executed successfully, and the output is as follows:\n%s", string(output))
	return string(output), nil
}

func resourceDelete(deployType string) (string, error) {
	cmd := exec.Command("dubboctl", "image", "deploy", deployType)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("Command execution failed: %s\nError: %v", string(output), err)
	}
	fmt.Printf("The command was executed successfully, and the output is as follows:\n%s", string(output))
	return string(output), nil
}
