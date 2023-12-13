package util

import (
	"math/rand"
)

// shamelessly adapted from https://stackoverflow.com/questions/71486991
func RandomString(length int) string {

	res := make([]rune, length)
	alphabet := []rune("abcdefghijklmnopqrstuvwxyz")

  for i := 0; i < length; i++ {
    res[i] = alphabet[rand.Intn(26)]
  }

  return string(res)
}
