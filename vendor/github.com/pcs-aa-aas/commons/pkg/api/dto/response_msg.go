package dto

type ResponseMsg struct {
	Status   float64     `json:"status"`
	ErrorMsg string      `json:"error_msg,omitempty"`
	Result   interface{} `json:"result,omitempty"`
}
