package env

import (
	"log"
	"os"
)

// Required returns the value of the named environment variable.
// It calls log.Fatalf if the variable is unset or empty.
func Required(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("%s is required", key)
	}
	return v
}
