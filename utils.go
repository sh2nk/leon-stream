package main

import (
	"math"
	"math/rand"
	"os"
	"time"
)

func randomInt32() int32 {
	rand.Seed(time.Now().UnixNano())
	return rand.Int31n(math.MaxInt32)
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
