package service

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func (s *Service) newAccessToken(username string) (string, int64, error) {
	if s.Config.Auth.JWTSecret == "" {
		return "", 0, errors.New("missing jwt secret")
	}
	expiresIn := s.Config.Auth.JWTExpireHours * 3600
	claims := jwt.RegisteredClaims{
		Subject:   username,
		Issuer:    "filehub",
		IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
		ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(time.Duration(s.Config.Auth.JWTExpireHours) * time.Hour)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(s.Config.Auth.JWTSecret))
	if err != nil {
		return "", 0, err
	}
	return signed, expiresIn, nil
}

func (s *Service) ParseAccessToken(tokenString string) (*jwt.RegisteredClaims, error) {
	if tokenString == "" {
		return nil, errors.New("missing token")
	}
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.Config.Auth.JWTSecret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

func (s *Service) NewPreviewToken(fileID string, ttl time.Duration) (string, error) {
	if s.Config.Auth.JWTSecret == "" {
		return "", errors.New("missing jwt secret")
	}
	claims := jwt.RegisteredClaims{
		Subject:   fileID,
		Issuer:    "filehub-preview",
		IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
		ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(ttl)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.Config.Auth.JWTSecret))
}

func (s *Service) ParsePreviewToken(tokenString string) (*jwt.RegisteredClaims, error) {
	if tokenString == "" {
		return nil, errors.New("missing token")
	}
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.Config.Auth.JWTSecret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	if claims.Issuer != "filehub-preview" {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}
