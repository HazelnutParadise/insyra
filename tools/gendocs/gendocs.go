package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

var (
	dirFlag    = flag.String("dir", "docs", "docs directory to scan")
	outputFlag = flag.String("output", "docs/_sidebar.md", "output _sidebar.md path")
	repoFlag   = flag.String("repo", "", "repo url to set in docs index (overrides GITHUB_REPOSITORY env)")
)

func main() {
	flag.Parse()
	entries := map[string][]string{}
	err := filepath.WalkDir(*dirFlag, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(d.Name()) != ".md" {
			return nil
		}
		// ignore files starting with underscore
		if strings.HasPrefix(d.Name(), "_") {
			return nil
		}
		rel, e := filepath.Rel(*dirFlag, path)
		if e != nil {
			return e
		}
		entries[filepath.Dir(rel)] = append(entries[filepath.Dir(rel)], rel)
		return nil
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "scan error: %v\n", err)
		os.Exit(1)
	}

	// Sort and write
	out, err := os.Create(*outputFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create output error: %v\n", err)
		os.Exit(1)
	}
	defer out.Close()

	w := bufio.NewWriter(out)
	defer w.Flush()

	// Homepage link to README.md if present
	rootReadme := filepath.Join(*dirFlag, "README.md")
	if _, err := os.Stat(rootReadme); err == nil {
		fmt.Fprintln(w, "* [Home](README.md)")
	}

	// Get sorted list of directories (root first)
	dirs := make([]string, 0, len(entries))
	for k := range entries {
		dirs = append(dirs, k)
	}
	sort.Slice(dirs, func(i, j int) bool {
		// root comes first
		if dirs[i] == "." {
			return true
		}
		if dirs[j] == "." {
			return false
		}
		return dirs[i] < dirs[j]
	})

	for _, d := range dirs {
		files := entries[d]
		sort.Strings(files)
		if d == "." {
			// top-level files (except README which is already added)
			for _, f := range files {
				if filepath.Base(f) == "README.md" {
					continue
				}
				title := titleFromFile(filepath.Join(*dirFlag, f))
				fmt.Fprintf(w, "* [%s](%s)\n", title, f)
			}
			continue
		}

		// Directory header
		dirName := filepath.Base(d)
		fmt.Fprintf(w, "* %s\n", dirName)
		// list files under it with indentation
		for _, f := range files {
			rel := f
			title := titleFromFile(filepath.Join(*dirFlag, f))
			fmt.Fprintf(w, "  * [%s](%s)\n", title, rel)
		}
	}

	fmt.Fprintf(os.Stderr, "generated %s\n", *outputFlag)

	// Also generate docs/index.html (Docsify entry) so index.html need not be tracked in repo
	if err := writeIndex(*dirFlag, *repoFlag); err != nil {
		fmt.Fprintf(os.Stderr, "generate index error: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "generated %s\n", filepath.Join(*dirFlag, "index.html"))
}

func writeIndex(dir, repoFlag string) error {
	// Prefer explicit repo flag, otherwise use GitHub Actions env var
	repoURL := ""
	if repoFlag != "" {
		repoURL = repoFlag
	} else if repo := os.Getenv("GITHUB_REPOSITORY"); repo != "" {
		repoURL = "https://github.com/" + repo
	}

	tmpl := `<!DOCTYPE html>
<html>
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>insyra Docs</title>
    <meta name="description" content="insyra documentation powered by Docsify" />
    <link rel="stylesheet" href="//unpkg.com/docsify/lib/themes/vue.css" />
  </head>
  <body>
    <div id="app"></div>

    <script>
      window.$docsify = {
        name: 'insyra',
        repo: '%s',
        loadSidebar: true
      };
    </script>

    <script src="//unpkg.com/docsify/lib/docsify.min.js"></script>
  </body>
</html>
`

	content := fmt.Sprintf(tmpl, repoURL)
	path := filepath.Join(dir, "index.html")
	return os.WriteFile(path, []byte(content), 0644)
}

func titleFromFile(path string) string {
	// try to find first heading in file
	f, err := os.Open(path)
	if err != nil {
		return nameToTitle(filepath.Base(path))
	}
	defer f.Close()
	r := bufio.NewReader(f)
	for {
		line, err := r.ReadString('\n')
		if err != nil && err != io.EOF {
			break
		}
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") {
			// remove leading # and spaces
			line = strings.TrimLeft(line, "#")
			line = strings.TrimSpace(line)
			if line != "" {
				// ensure first rune is uppercase
				runes := []rune(line)
				runes[0] = unicode.ToUpper(runes[0])
				return string(runes)
			}
		}
		if err == io.EOF {
			break
		}
	}
	return nameToTitle(filepath.Base(path))
}

func nameToTitle(name string) string {
	base := strings.TrimSuffix(name, filepath.Ext(name))
	base = strings.ReplaceAll(base, "_", " ")
	// If filename already contains uppercase letters (e.g., DataTable), preserve original casing
	for _, r := range base {
		if unicode.IsUpper(r) {
			return base
		}
	}
	// Otherwise capitalize only the first rune (e.g., "parquet" -> "Parquet")
	if base == "" {
		return base
	}
	runes := []rune(base)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}
