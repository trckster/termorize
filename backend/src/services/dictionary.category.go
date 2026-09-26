package services

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"gorm.io/gorm"
	"termorize/src/models"
)

type idiomCategorySource struct{ Language, Category, ProverbCategory string }

func categorySource(edition string) *idiomCategorySource {
	switch edition {
	case "ukwiktionary":
		return &idiomCategorySource{"uk", "Категорія:Фразеологізми/uk", "Категорія:Прислів’я/uk"}
	case "enwiktionary-es":
		return &idiomCategorySource{"es", "Category:Spanish idioms", "Category:Spanish proverbs"}
	case "enwiktionary-pt":
		return &idiomCategorySource{"pt", "Category:Portuguese idioms", "Category:Portuguese proverbs"}
	}
	return nil
}

// These sources expose explicit idiom categories through MediaWiki rather than
// a usable classified extract. Normalize them into the existing import pipeline.
func (w *DictionaryWorker) downloadCategoryIdioms(ctx context.Context, conn *gorm.DB, job *models.DictionaryImportJob, file *os.File) error {
	endpoint, err := url.Parse(job.DownloadURL)
	if err != nil {
		return err
	}
	compressed := gzip.NewWriter(file)
	defer compressed.Close()
	encoder := json.NewEncoder(compressed)
	source := categorySource(job.Edition)
	if source == nil {
		return ErrDictionaryEdition
	}
	continuation := map[string]string{}
	seen := map[string]bool{}
	for page := 0; page < 100; page++ {
		query := endpoint.Query()
		query.Set("action", "query")
		query.Set("format", "json")
		query.Set("formatversion", "2")
		query.Set("generator", "categorymembers")
		query.Set("gcmtitle", source.Category)
		query.Set("gcmnamespace", "0")
		query.Set("gcmtype", "page")
		query.Set("gcmlimit", "500")
		query.Set("prop", "info|categories")
		query.Set("clcategories", source.ProverbCategory)
		query.Set("cllimit", "500")
		for key, value := range continuation {
			query.Set(key, value)
		}
		endpoint.RawQuery = query.Encode()
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
		if err != nil {
			return err
		}
		request.Header.Set("User-Agent", "Termorize/1.0 (https://github.com/trckster/termorize; idiom importer)")
		response, err := w.Client.Do(request)
		if err != nil {
			return fmt.Errorf("download category: %w", err)
		}
		body, readErr := io.ReadAll(io.LimitReader(response.Body, (5<<20)+1))
		response.Body.Close()
		if readErr != nil {
			return readErr
		}
		if response.StatusCode != http.StatusOK {
			return fmt.Errorf("category returned HTTP %d", response.StatusCode)
		}
		if len(body) > 5<<20 {
			return errors.New("category response exceeds 5 MiB")
		}
		var result struct {
			Error *struct {
				Code string `json:"code"`
			} `json:"error"`
			Query *struct {
				Pages []struct {
					Title      string `json:"title"`
					Namespace  int    `json:"ns"`
					Redirect   bool   `json:"redirect"`
					Categories []struct {
						Title string `json:"title"`
					} `json:"categories"`
				} `json:"pages"`
			} `json:"query"`
			Continue map[string]string `json:"continue"`
		}
		if err := json.Unmarshal(body, &result); err != nil {
			return err
		}
		if result.Error != nil {
			return fmt.Errorf("category API error: %s", result.Error.Code)
		}
		if result.Query == nil || result.Query.Pages == nil {
			return errors.New("category API response missing members")
		}
		job.DownloadedBytes += int64(len(body))
		if err := conn.Save(job).Error; err != nil {
			return err
		}
		for _, member := range result.Query.Pages {
			if member.Namespace != 0 || member.Redirect || len(member.Categories) > 0 {
				continue
			}
			if err := encoder.Encode(map[string]any{"word": member.Title, "lang_code": source.Language, "pos": "phrase", "tags": []string{"idiomatic"}}); err != nil {
				return err
			}
		}
		continuation = result.Continue
		if len(continuation) == 0 {
			return compressed.Close()
		}
		token, _ := json.Marshal(continuation)
		if seen[string(token)] {
			return errors.New("category API repeated continuation token")
		}
		seen[string(token)] = true
		// Drop previous continuation keys; MediaWiki may change the continuation module.
		endpoint.RawQuery = ""
	}
	return errors.New("category exceeds 100-page limit")
}
