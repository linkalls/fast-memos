package auth

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// 仮のシークレットキー。本番環境では設定ファイルなどから読み込むべきです。
const jwtSecret = "your-secret-key"

// HashPassword はパスワードをハッシュ化します
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

// CheckPasswordHash はハッシュ化されたパスワードと平文のパスワードを比較します
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateJWT はユーザーIDを含むJWTを生成します
func GenerateJWT(userID uint) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Hour * 72).Unix(), // トークンの有効期限 (例: 72時間)
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(jwtSecret))
}

// ValidateJWT はJWTを検証し、ユーザーIDを返します
func ValidateJWT(tokenString string) (uint, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(jwtSecret), nil
	})

	if err != nil {
		return 0, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		userID, ok := claims["user_id"].(float64) // JWTの数値はfloat64としてデコードされる
		if !ok {
			return 0, errors.New("user_id claim is not valid")
		}
		return uint(userID), nil
	}

	return 0, errors.New("invalid token")
}

// AuthMiddleware はJWTを検証するFiberミドルウェアです
func AuthMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Missing or malformed JWT"})
		}

		// "Bearer <token>" 形式を想定
		const BearerSchema = "Bearer "
		if len(authHeader) <= len(BearerSchema) || authHeader[:len(BearerSchema)] != BearerSchema {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Malformed token"})
		}
		tokenString := authHeader[len(BearerSchema):]

		userID, err := ValidateJWT(tokenString)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid or expired JWT", "details": err.Error()})
		}

		c.Locals("userID", userID) // 後続のハンドラでユーザーIDを使用できるようにする
		return c.Next()
	}
}
