package models

import "time"

// User คือข้อมูล user จาก AD
type User struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Department  string `json:"department"`
	Role        string `json:"role"` // admin, manager, employee
}

// LoginRequest คือ request สำหรับ login
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse คือ response หลัง login สำเร็จ
type LoginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      User      `json:"user"`
}

// Claims คือข้อมูลที่เก็บใน JWT
type Claims struct {
	UserID      string `json:"user_id"`
	Username    string `json:"username"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Department  string `json:"department"`
	Role        string `json:"role"`
}