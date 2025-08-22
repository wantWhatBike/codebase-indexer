package response

import "fmt"

type codeMsg struct {
	Code    string
	Message string
}

func (c *codeMsg) Error() string {
	return fmt.Sprintf("code: %s, message: %s", c.Code, c.Message)
}

// NewError creates a new codeMsg.
func NewError(code string, msg string) error {
	return &codeMsg{Code: code, Message: msg}
}
