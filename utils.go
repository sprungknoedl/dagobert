package main

import (
	"encoding/json"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

func GetUsername(c *gin.Context) string {
	session := sessions.Default(c)
	claims := session.Get("oidcClaims").(string)

	obj := map[string]interface{}{}
	err := json.Unmarshal([]byte(claims), &obj)
	if err != nil {
		return "unknown"
	}

	if email, ok := obj["email"].(string); ok {
		return email
	}
	return "unknown"
}
