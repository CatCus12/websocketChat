package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

type Claims struct {
	Username string `json:"username"`
	Password string `json:"password"`
	jwt.RegisteredClaims
}

func GenerateJWT(username, password string, jwtSecretKey []byte) (string, error) {
	// Получаем текущее время и время истечения токена
	now := time.Now()
	expirationTime := now.Add(24 * time.Hour) // срок действия токена — 24 часа

	// Создаём claims (данные, которые будут храниться в токене)
	claims := &Claims{
		Username: username,
		Password: password,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "your-app", // Приложение или сервис, который создал токен
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	// Создаём новый токен с использованием HMAC SHA256
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signedToken, err := token.SignedString(jwtSecretKey)
	if err != nil {
		return "", fmt.Errorf("не удалось подписать токен: %v", err)
	}

	return signedToken, nil
}

// Функция для проверки и парсинга токена
func ValidateJWT(signedToken string) (*Claims, error) {
	// Парсим токен
	token, err := jwt.ParseWithClaims(signedToken, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Проверка метода подписания
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("неподдерживаемый метод подписания: %v", token.Header["alg"])
		}
		return jwtSecretKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("не удалось распарсить токен: %v", err)
	}

	// Проверка токена на валидность и срок годности
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		// Токен действителен, возвращаем данные
		return claims, nil
	} else {
		return nil, fmt.Errorf("невалидный токен")
	}
}
func requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Получаем токен из заголовка Authorization
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Токен должен быть в формате "Bearer <token>"
		tokenString := authHeader[7:]

		// Проверка и валидация токена
		claims, err := ValidateJWT(tokenString)
		if err != nil {
			log.Printf("Ошибка валидации токена: %v", err)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Если токен валиден, передаем управление следующему обработчику
		fmt.Println("Авторизованный пользователь:", claims.Username)
		next(w, r)
	})
}
