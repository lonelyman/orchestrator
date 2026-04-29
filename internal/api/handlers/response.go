package handlers

import "github.com/gofiber/fiber/v3"

// Response คือ standard response format ของบริษัท
type Response struct {
	Data  interface{} `json:"data,omitempty"`
	Error interface{} `json:"error,omitempty"`
}

// ErrorDetail คือรายละเอียดของ error
type ErrorDetail struct {
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

// OK ส่ง response สำเร็จ
func OK(c fiber.Ctx, data interface{}) error {
	return c.JSON(Response{Data: data})
}

// Fail ส่ง response error
func Fail(c fiber.Ctx, status int, message string, code string) error {
	return c.Status(status).JSON(Response{
		Error: ErrorDetail{
			Message: message,
			Code:    code,
		},
	})
}
