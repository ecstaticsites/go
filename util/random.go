package util

import (
	"fmt"
	"math/rand"
)

func RandomConsonant() rune {
	return []rune("bcdfghjklmnpqrstvwxz")[rand.Intn(20)]
}

func RandomVowel() rune {
	return []rune("aeiouy")[rand.Intn(6)]
}

func RandomIam() string {
	c1 := string(RandomConsonant())
	v1 := string(RandomVowel())
	c2 := string(RandomConsonant())
	v2 := string(RandomVowel())
	return fmt.Sprintf("%s%s%s%s", c1, v1, c2, v2)
}

func RandomIamTriple() string {
	i1 := RandomIam()
	i2 := RandomIam()
	i3 := RandomIam()
	return fmt.Sprintf("%s-%s-%s", i1, i2, i3)
}
