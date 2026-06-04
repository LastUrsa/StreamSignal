package runtimeenv

import (
	"os"
	"strings"
	"unicode"
)

const (
	EnvironmentVariable = "STREAMSIGNAL_ENV"
	DataDirVariable     = "STREAMSIGNAL_DATA_DIR"
)

func Name() string {
	value := normalize(os.Getenv(EnvironmentVariable))
	if value == "" || value == "prod" || value == "production" || value == "release" {
		return "production"
	}
	return value
}

func IsProduction() bool {
	return Name() == "production"
}

func DataDirName() string {
	if IsProduction() {
		return "StreamSignal"
	}
	return "StreamSignal-" + Name()
}

func SecretNamespace() string {
	if IsProduction() {
		return "streamsignal"
	}
	return "streamsignal-" + Name()
}

func normalize(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	lastDash := false
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(r)
			lastDash = false
			continue
		}
		if (r == '-' || r == '_' || unicode.IsSpace(r)) && !lastDash && builder.Len() > 0 {
			builder.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(builder.String(), "-")
}
