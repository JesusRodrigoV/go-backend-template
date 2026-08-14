package services

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/JesusRodrigoV/granBackend/internal/core/domain"
	"github.com/JesusRodrigoV/granBackend/pkg/utils"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type googleUser struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
}

func (s *AuthService) getGoogleConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     s.cfg.GoogleClientID,
		ClientSecret: s.cfg.GoogleClientSecret,
		RedirectURL:  s.cfg.GoogleRedirectURL,
		Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile"},
		Endpoint:     google.Endpoint,
	}
}

func (s *AuthService) GetGoogleAuthURL() string {
	return s.getGoogleConfig().AuthCodeURL("state_random_string", oauth2.AccessTypeOffline)
}

func (s *AuthService) LoginWithGoogle(code, ip, userAgent string) (string, string, bool, error) {
	ctx := context.Background()
	conf := s.getGoogleConfig()

	token, err := conf.Exchange(ctx, code)
	if err != nil {
		return "", "", false, errors.New("falló el intercambio de código con google")
	}

	client := conf.Client(ctx, token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return "", "", false, errors.New("falló al obtener datos del usuario")
	}
	defer resp.Body.Close()

	var gUser googleUser
	if err := json.NewDecoder(resp.Body).Decode(&gUser); err != nil {
		return "", "", false, err
	}

	var user *domain.User
	var userID uuid.UUID

	social, err := s.repo.FindSocial("google", gUser.ID)

	if err == nil {
		userID = social.UserID
		user, err = s.repo.GetByEmail(gUser.Email)
		if err != nil {
			return "", "", false, errors.New("error de integridad: usuario social no encontrado en users")
		}
	} else {
		user, err = s.repo.GetByEmail(gUser.Email)

		if err == nil {
			userID = user.ID
			s.linkSocial(userID, "google", gUser.ID, gUser)
		} else {
			newUser := &domain.User{
				ID:       uuid.New(),
				Email:    gUser.Email,
				Verified: true,
				Password: "",
			}
			if err := s.repo.Save(newUser); err != nil {
				return "", "", false, err
			}

			user = newUser
			userID = newUser.ID
			s.linkSocial(userID, "google", gUser.ID, gUser)
		}
	}

	if user.Mfa {
		s.logEvent(userID, "LOGIN_GOOGLE_MFA_CHALLENGE", ip, userAgent, nil)

		tempToken, err := utils.GenerateTempToken(userID.String(), s.jwtSecret)
		if err != nil {
			return "", "", false, err
		}

		return "", tempToken, true, nil
	}

	accessToken, err := utils.GenerateJWT(userID.String(), s.jwtSecret)
	if err != nil {
		return "", "", false, err
	}

	refreshTokenRaw, _ := utils.GenerateRandomString(32)
	refreshTokenHash := utils.HashToken(refreshTokenRaw)

	newSession := &domain.Session{
		ID:        uuid.New(),
		UserID:    userID,
		Hash:      refreshTokenHash,
		Device:    userAgent,
		IP:        ip,
		Exp:       time.Now().Add(time.Hour * 24 * 30),
		CreatedAt: time.Now(),
	}
	s.repo.SaveSession(newSession)

	s.logEvent(userID, "LOGIN_GOOGLE_SUCCESS", ip, userAgent, nil)

	return accessToken, refreshTokenRaw, false, nil
}

func (s *AuthService) linkSocial(userID uuid.UUID, provider, provID string, rawData any) {
	rawMap := make(map[string]any)
	jsonRaw, _ := json.Marshal(rawData)
	json.Unmarshal(jsonRaw, &rawMap)

	s.repo.SaveSocial(&domain.Social{
		UserID:   userID,
		Provider: provider,
		ProvID:   provID,
		Raw:      rawMap,
	})
}
