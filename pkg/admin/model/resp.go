package model

const (
	successCode = 200
	errorCode   = 500
)

type CommonResp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data"`
}

func (r *CommonResp) WithCode(code int) *CommonResp {
	r.Code = code
	return r
}

func (r *CommonResp) WithMsg(msg string) *CommonResp {
	r.Msg = msg
	return r
}

func (r *CommonResp) WithData(data any) *CommonResp {
	r.Data = data
	return r
}

func NewSuccessResp(data any) *CommonResp {
	return &CommonResp{
		Code: successCode,
		Msg:  "success",
		Data: data,
	}
}

func NewErrorResp(msg string) *CommonResp {
	return &CommonResp{
		Code: errorCode,
		Msg:  msg,
		Data: nil,
	}
}
