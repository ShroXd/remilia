package remilia

func GetOrDefault(s *string, def string) string {
	if *s == "" {
		return def
	}

	return *s
}
