package utils

// MaskToken masks the token
func MaskToken(token string) string {
	if token == "" {
		return ""
	}

	if len(token) <= 8 {
		return "***"
	}

	return token[:4] + "..." + token[len(token)-4:]
}
