package aionui

import (
	"crypto/hmac"
	"crypto/sha256"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	apmodel "github.com/QuantumNous/new-api/model/agentplatform"
	"github.com/golang-jwt/jwt/v5"
)

func normalizeClientUpdateAccessInput(input ClientUpdateAccessInput) ClientUpdateAccessInput {
	input.Platform = strings.TrimSpace(strings.ToLower(input.Platform))
	input.CurrentVersion = strings.TrimSpace(input.CurrentVersion)
	input.ExpectedVersion = strings.TrimSpace(input.ExpectedVersion)
	return input
}

func clientUpdateCapabilitySigningKey() []byte {
	mac := hmac.New(sha256.New, []byte(common.CryptoSecret))
	_, _ = mac.Write([]byte("aionui/client-update-capability/v1"))
	return mac.Sum(nil)
}

func issueClientUpdateArtifactCapability(userID int, deviceID string, pkg apmodel.ClientPackage, now time.Time) (string, time.Time, error) {
	if userID <= 0 || pkg.Id <= 0 || !validClientPackagePlatform(pkg.Platform) || !validClientPackageVersion(pkg.Version) || strings.TrimSpace(deviceID) == "" || common.CryptoSecret == "" {
		return "", time.Time{}, ErrClientUpdateCapabilityInvalid
	}
	files := []string{pkg.FileName}
	if strings.TrimSpace(pkg.UpdateFileName) != "" {
		files = append(files, pkg.UpdateFileName)
	}
	for _, fileName := range files {
		if strings.TrimSpace(fileName) == "" {
			return "", time.Time{}, ErrClientUpdateCapabilityInvalid
		}
	}
	sort.Strings(files)
	now = now.UTC()
	expiresAt := now.Add(clientUpdateCapabilityTTL)
	claims := clientUpdateArtifactClaims{
		Purpose:         clientUpdateCapabilityPurpose,
		UserId:          userID,
		DeviceId:        strings.TrimSpace(deviceID),
		ClientPackageId: pkg.Id,
		Platform:        pkg.Platform,
		Version:         pkg.Version,
		Files:           files,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    clientUpdateCapabilityIssuer,
			Audience:  jwt.ClaimStrings{clientUpdateCapabilityAudience},
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			NotBefore: jwt.NewNumericDate(now.Add(-5 * time.Second)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(clientUpdateCapabilitySigningKey())
	if err != nil {
		return "", time.Time{}, err
	}
	return token, expiresAt, nil
}

func parseClientUpdateArtifactCapability(raw string, now time.Time) (*clientUpdateArtifactClaims, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || common.CryptoSecret == "" {
		return nil, ErrClientUpdateCapabilityInvalid
	}
	claims := &clientUpdateArtifactClaims{}
	parsed, err := jwt.ParseWithClaims(raw, claims, func(token *jwt.Token) (any, error) {
		if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, errors.New("unexpected signing method")
		}
		return clientUpdateCapabilitySigningKey(), nil
	}, jwt.WithTimeFunc(func() time.Time { return now }), jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithIssuer(clientUpdateCapabilityIssuer), jwt.WithAudience(clientUpdateCapabilityAudience), jwt.WithExpirationRequired(), jwt.WithIssuedAt(), jwt.WithLeeway(5*time.Second))
	if err != nil || parsed == nil || !parsed.Valid || claims.Purpose != clientUpdateCapabilityPurpose || claims.UserId <= 0 || claims.ClientPackageId <= 0 || strings.TrimSpace(claims.DeviceId) == "" || !validClientPackagePlatform(claims.Platform) || !validClientPackageVersion(claims.Version) || len(claims.Files) == 0 {
		return nil, ErrClientUpdateCapabilityInvalid
	}
	for _, fileName := range claims.Files {
		if strings.TrimSpace(fileName) == "" {
			return nil, ErrClientUpdateCapabilityInvalid
		}
	}
	return claims, nil
}

func clientUpdateCapabilityAllowsFile(files []string, fileName string) bool {
	for _, allowed := range files {
		if allowed == fileName {
			return true
		}
	}
	return false
}
