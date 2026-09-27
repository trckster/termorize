package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"termorize/src/enums"
	"termorize/src/integrations/jev"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	client := jev.NewClient(os.Getenv("OPENROUTER_API_KEY"))
	for _, sample := range []struct {
		word     string
		language enums.Language
		want     enums.PartOfSpeech
	}{
		{"io", enums.LanguageIt, enums.PartOfSpeechPronoun},
		{"spill the beans", enums.LanguageEn, enums.PartOfSpeechPhrase},
		{"well-known", enums.LanguageEn, enums.PartOfSpeechAdjective},
		{"the", enums.LanguageEn, enums.PartOfSpeechUnknown},
	} {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		category, err := client.Classify(ctx, sample.word, sample.language)
		cancel()
		if err != nil {
			log.Fatal(err)
		}
		if category != sample.want && category != enums.PartOfSpeechUnknown {
			log.Fatalf("%s %q: got %s, expected %s", sample.language, sample.word, category, sample.want)
		}
		fmt.Printf("%s %q: %s\n", sample.language, sample.word, category)
	}
}
