// Command docBundle embeds the repository's documentation without runtime filesystem access.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type document struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

func main() {
	err := generate()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)

		os.Exit(1)
	}
}

func generate() error {
	var documents []document

	for _, group := range []string{"docs", "design"} {
		paths, err := filepath.Glob(filepath.Join("../../..", group, "*.md"))
		if err != nil {
			return err
		}

		for _, path := range paths {
			//nolint:gosec // Paths come only from the fixed repository documentation glob.
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			documents = append(documents, document{Path: group + "/" + filepath.Base(path), Content: string(data)})
		}
	}

	data, err := json.MarshalIndent(documents, "", "  ")
	if err != nil {
		return err
	}

	const permission = 0o644

	return os.WriteFile("documents.json", append(data, '\n'), permission)
}
