package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	// InitDataHeader is the header name for Telegram init data
	InitDataHeader = "X-Telegram-Init-Data"
	// UserContextKey is the key for user data in context
	UserContextKey = "user"
)

// InitData represents parsed Telegram init data
type InitData struct {
	User     *TelegramUser `json:"user"`
	Chat     *TelegramChat `json:"chat,omitempty"`
	AuthDate int64         `json:"auth_date"`
	Hash     string        `json:"hash"`
	QueryID  string        `json:"query_id,omitempty"`
}

// TelegramUser represents a user in init data
type TelegramUser struct {
	ID           int64  `json:"id"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name,omitempty"`
	Username     string `json:"username,omitempty"`
	LanguageCode string `json:"language_code,omitempty"`
	IsBot        bool   `json:"is_bot,omitempty"`
}

// TelegramChat represents a chat in init data
type TelegramChat struct {
	ID       int64  `json:"id"`
	Type     string `json:"type,omitempty"`
	Title    string `json:"title,omitempty"`
	Username string `json:"username,omitempty"`
}

// AuthMiddleware creates a middleware for Telegram InitData authorization
func AuthMiddleware(botToken string) gin.HandlerFunc {
	return func(c *gin.Context) {
		initDataStr := c.GetHeader(InitDataHeader)
		if initDataStr == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "missing init data",
			})
			return
		}

		initData, err := ParseInitData(initDataStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": fmt.Sprintf("invalid init data: %v", err),
			})
			return
		}

		// Validate hash
		if !ValidateInitData(initDataStr, botToken) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid init data hash",
			})
			return
		}

		// Check auth date (not older than 24 hours)
		authTime := time.Unix(initData.AuthDate, 0)
		if time.Since(authTime) > 24*time.Hour {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "init data expired",
			})
			return
		}

		// Set user data in context
		c.Set(UserContextKey, initData.User)
		c.Next()
	}
}

// HRAdminMiddleware ensures the user has HR or Manager role
// This requires additional context from the database, so it should be used after AuthMiddleware
func HRAdminMiddleware(getUserRole func(telegramID int64) (string, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, exists := c.Get(UserContextKey)
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "user not found in context",
			})
			return
		}

		tgUser := user.(*TelegramUser)
		role, err := getUserRole(tgUser.ID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": "failed to check role",
			})
			return
		}

		if role != "hr" && role != "manager" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "insufficient permissions",
			})
			return
		}

		c.Next()
	}
}

// ParseInitData parses Telegram init data string into InitData struct
func ParseInitData(data string) (*InitData, error) {
	parts := strings.Split(data, "&")
	params := make(map[string]string)

	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		key, _ := urlDecode(kv[0])
		value, _ := urlDecode(kv[1])
		params[key] = value
	}

	hash := params["hash"]
	delete(params, "hash")

	authDate, _ := strconv.ParseInt(params["auth_date"], 10, 64)
	delete(params, "auth_date")

	queryID := params["query_id"]
	delete(params, "query_id")

	var user *TelegramUser
	if userData, ok := params["user"]; ok {
		user = &TelegramUser{}
		if err := json.Unmarshal([]byte(userData), user); err != nil {
			return nil, fmt.Errorf("failed to parse user: %w", err)
		}
	}

	var chat *TelegramChat
	if chatData, ok := params["chat"]; ok {
		chat = &TelegramChat{}
		if err := json.Unmarshal([]byte(chatData), chat); err != nil {
			return nil, fmt.Errorf("failed to parse chat: %w", err)
		}
	}

	return &InitData{
		User:     user,
		Chat:     chat,
		AuthDate: authDate,
		Hash:     hash,
		QueryID:  queryID,
	}, nil
}

// ValidateInitData validates the hash of init data
func ValidateInitData(data string, botToken string) bool {
	parts := strings.Split(data, "&")
	params := make([]string, 0, len(parts))

	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 || kv[0] == "hash" {
			continue
		}
		key, _ := urlDecode(kv[0])
		value, _ := urlDecode(kv[1])
		params = append(params, fmt.Sprintf("%s=%s", key, value))
	}

	// Sort parameters
	dataCheckString := strings.Join(params, "\n")

	// Create secret key
	secretKey := hmacSha256([]byte("WebAppData"), []byte(botToken))

	// Create HMAC-SHA256
	mac := hmac.New(sha256.New, secretKey)
	mac.Write([]byte(dataCheckString))
	calculatedHash := hexEncode(mac.Sum(nil))

	// Get provided hash
	var providedHash string
	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 && kv[0] == "hash" {
			providedHash, _ = urlDecode(kv[1])
			break
		}
	}

	return hmac.Equal([]byte(calculatedHash), []byte(providedHash))
}

// urlDecode decodes URL-encoded string
func urlDecode(s string) (string, error) {
	result, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		result = []byte(s)
	}
	return string(result), nil
}

// hexEncode encodes bytes to hex string
func hexEncode(data []byte) string {
	const hexChars = "0123456789abcdef"
	result := make([]byte, len(data)*2)
	for i, b := range data {
		result[i*2] = hexChars[b>>4]
		result[i*2+1] = hexChars[b&0x0f]
	}
	return string(result)
}

// hmacSha256 creates HMAC-SHA256
func hmacSha256(key, data []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(data)
	return mac.Sum(nil)
}

// GetUserFromContext retrieves user from gin context
func GetUserFromContext(c *gin.Context) (*TelegramUser, error) {
	user, exists := c.Get(UserContextKey)
	if !exists {
		return nil, fmt.Errorf("user not found in context")
	}
	tgUser, ok := user.(*TelegramUser)
	if !ok {
		return nil, fmt.Errorf("invalid user type in context")
	}
	return tgUser, nil
}

// GetUserIDFromContext retrieves user UUID from context (requires DB lookup)
func GetUserIDFromContext(c *gin.Context, getUserByTelegramID func(telegramID int64) (uuid.UUID, error)) (uuid.UUID, error) {
	user, err := GetUserFromContext(c)
	if err != nil {
		return uuid.Nil, err
	}
	return getUserByTelegramID(user.ID)
}
