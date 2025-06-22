package handlers

import (
	"github.com/linkalls/fast-memos/auth"
	"github.com/linkalls/fast-memos/database"
	"github.com/linkalls/fast-memos/models"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type RegisterUserInput struct {
	Username string `json:"username" xml:"username" form:"username" validate:"required,min=3"`
	Password string `json:"password" xml:"password" form:"password" validate:"required,min=6"`
}

type LoginUserInput struct {
	Username string `json:"username" xml:"username" form:"username" validate:"required"`
	Password string `json:"password" xml:"password" form:"password" validate:"required"`
}

func RegisterUser(c *fiber.Ctx) error {
	input := new(RegisterUserInput)

	if err := c.BodyParser(input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Cannot parse JSON", "details": err.Error()})
	}

	// TODO: ここでバリデーションライブラリ (例: go-playground/validator) を使うとより良い
	if len(input.Username) < 3 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Username must be at least 3 characters long"})
	}
	if len(input.Password) < 6 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Password must be at least 6 characters long"})
	}

	hashedPassword, err := auth.HashPassword(input.Password)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not hash password", "details": err.Error()})
	}

	user := models.User{
		Username: input.Username,
		Password: hashedPassword,
	}

	result := database.DB.Create(&user)
	if result.Error != nil {
		// GORM v2では重複エラーは IsDuplicatedError ではなく、エラーメッセージやコードで判断することが多い
		// SQLiteの場合、"UNIQUE constraint failed: users.username" のようなエラーメッセージになる
		// ここでは簡略化のため、一般的なエラーとして処理
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not create user", "details": result.Error.Error()})
	}

	// パスワードを含まないユーザー情報を返す
	userResponse := fiber.Map{
		"id":       user.ID,
		"username": user.Username,
	}

	return c.Status(fiber.StatusCreated).JSON(userResponse)
}

func LoginUser(c *fiber.Ctx) error {
	input := new(LoginUserInput)

	if err := c.BodyParser(input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Cannot parse JSON", "details": err.Error()})
	}

	var user models.User
	if err := database.DB.Where("username = ?", input.Username).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid username or password"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Database error", "details": err.Error()})
	}

	if !auth.CheckPasswordHash(input.Password, user.Password) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid username or password"})
	}

	token, err := auth.GenerateJWT(user.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not generate token", "details": err.Error()})
	}

	return c.JSON(fiber.Map{"token": token})
}
