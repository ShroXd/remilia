package remilia

import "regexp"

func GetOrDefault(s *string, def string) string {
	if s == nil || *s == "" {
		return def
	}

	return *s
}

func URLMatcher() func(s string) bool {
	urlPattern := `^(https?|ftp)://[^\s/$.?#].[^\s]*$`
	urlRegex, _ := regexp.Compile(urlPattern)

	return func(s string) bool {
		return urlRegex.MatchString(s)
	}
}
