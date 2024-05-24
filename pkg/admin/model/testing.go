package model

type MethodDetail struct {
	InputT     []interface{} `json:"parameterTypes"`
	ReturnType string        `json:"returnType"`
}

type MethodDescribe struct {
	InputType      string `json:"inputType"`
	InputDescribe  string `json:"inputDescribe"`
	OutputType     string `json:"outputType"`
	OutputDescribe string `json:"outputDescribe"`
}
