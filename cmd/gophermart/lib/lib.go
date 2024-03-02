package lib

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var secret = "PaBjK!7K$&qMUMTb"

type JWTClaims struct {
	jwt.RegisteredClaims
	UserID uuid.UUID `json:"user_id"`
}

func CreateJWT(userUUID uuid.UUID) (string, error) {
	claims := JWTClaims{
		UserID: userUUID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 24)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	jwtString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}

	return jwtString, nil
}

func CheckJWT(tokenString string) (uuid.UUID, error) {
	claims := JWTClaims{}
	token, err := jwt.ParseWithClaims(tokenString, &claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return uuid.Nil, err
	}

	if !token.Valid {
		return uuid.Nil, err
	}

	return claims.UserID, nil
}

func CalculateLuhn(number int64) int64 {
	checkNumber := checksum(number)

	if checkNumber == 0 {
		return 0
	}
	return 10 - checkNumber
}

func LuhnValid(number int64) bool {
	return (number%10+checksum(number/10))%10 == 0
}

func checksum(number int64) int64 {
	var luhn int64

	for i := 0; number > 0; i++ {
		var cur int64
		cur = number % 10

		if i%2 == 0 { // even
			cur = cur * 2
			if cur > 9 {
				cur = cur%10 + cur/10
			}
		}

		luhn += cur
		number = number / 10
	}
	return luhn % 10
}

func IsTableExist(p *pgxpool.Pool, table string) bool {
	var n int

	err := p.QueryRow(context.Background(), "SELECT 1 FROM information_schema.tables WHERE table_name = $1", table).Scan(&n)

	return err == nil
}

func RoundFloat(val float64, precision uint) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(val*ratio) / ratio
}
