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
	dirFlag    = flag.String("dir", "", "docs directory to scan (required)")
	outputFlag = flag.String("output", "", "output _sidebar.md path (required)")
	repoFlag   = flag.String("repo", "", "repo url to set in docs index (required)")
)

func main() {
	flag.Parse()

	// Required flags — fail fast if any are missing
	if *dirFlag == "" {
		fmt.Fprintln(os.Stderr, "ERROR: --dir is required")
		os.Exit(1)
	}
	if *outputFlag == "" {
		fmt.Fprintln(os.Stderr, "ERROR: --output is required")
		os.Exit(1)
	}
	if *repoFlag == "" {
		fmt.Fprintln(os.Stderr, "ERROR: --repo is required")
		os.Exit(1)
	}

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
	defer func() {
		if err := out.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "close output error: %v\n", err)
		}
	}()

	w := bufio.NewWriter(out)
	defer func() {
		if err := w.Flush(); err != nil {
			fmt.Fprintf(os.Stderr, "flush error: %v\n", err)
		}
	}()

	// Homepage link to README.md if present
	rootReadme := filepath.Join(*dirFlag, "README.md")
	if _, err := os.Stat(rootReadme); err == nil {
		if _, err := fmt.Fprintln(w, "* [Home](README.md)"); err != nil {
			fmt.Fprintf(os.Stderr, "write error: %v\n", err)
			os.Exit(1)
		}
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
				if _, err := fmt.Fprintf(w, "* [%s](%s)\n", title, f); err != nil {
					fmt.Fprintf(os.Stderr, "write error: %v\n", err)
					os.Exit(1)
				}
			}
			continue
		}

		// Directory header
		dirName := filepath.Base(d)
		if _, err := fmt.Fprintf(w, "* %s\n", dirName); err != nil {
			fmt.Fprintf(os.Stderr, "write error: %v\n", err)
			os.Exit(1)
		}
		// list files under it with indentation
		for _, f := range files {
			rel := f
			title := titleFromFile(filepath.Join(*dirFlag, f))
			if _, err := fmt.Fprintf(w, "  * [%s](%s)\n", title, rel); err != nil {
				fmt.Fprintf(os.Stderr, "write error: %v\n", err)
				os.Exit(1)
			}
		}
	}

	fmt.Fprintf(os.Stderr, "generated %s\n", *outputFlag)

	// Also generate docs/index.html (Docsify entry) so index.html need not be tracked in repo
	if err := writeIndex(*dirFlag, *repoFlag); err != nil {
		fmt.Fprintf(os.Stderr, "generate index error: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "generated %s\n", filepath.Join(*dirFlag, "index.html"))

	// Generate a navbar with an "Official site" link and a theme CSS with deep-blue colors
	siteURL := detectSiteURL(*dirFlag, *repoFlag)
	if err := writeNavbar(*dirFlag, siteURL); err != nil {
		fmt.Fprintf(os.Stderr, "generate navbar error: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "generated %s\n", filepath.Join(*dirFlag, "_navbar.md"))

	if err := writeThemeCSS(*dirFlag); err != nil {
		fmt.Fprintf(os.Stderr, "generate css error: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "generated %s\n", filepath.Join(*dirFlag, "docs.css"))

	if err := copyLogo(*dirFlag); err != nil {
		fmt.Fprintf(os.Stderr, "copy logo error: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "copied %s\n", filepath.Join(*dirFlag, "logo.webp"))
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
    <title>Insyra Docs</title>
    <meta name="description" content="Insyra documentation powered by Docsify" />
    <link rel="stylesheet" href="//unpkg.com/docsify/lib/themes/vue.css" />
    <!-- Custom theme overrides -->
    <link rel="stylesheet" href="docs.css" />
  </head>
  <body>
    <div id="app"></div>

    <script>
      window.$docsify = {
        name: 'Insyra',
        repo: '%s',
        loadSidebar: true,
        loadNavbar: true,
        logo: 'logo.webp'
      };
    </script>

    <script src="//unpkg.com/docsify/lib/docsify.min.js"></script>
	<script src="//unpkg.com/docsify-copy-code"></script>
	<script src="//cdn.jsdelivr.net/npm/prismjs@1.22/components/prism-go.min.js"></script>
  </body>
</html>
`

	content := fmt.Sprintf(tmpl, repoURL)
	path := filepath.Join(dir, "index.html")
	return os.WriteFile(path, []byte(content), 0644)
}

// detectSiteURL tries to find the official website URL from a docs config, OFFICIAL_SITE env, README, or repo
func detectSiteURL(dir, repoFlag string) string {
	// OFFICIAL_SITE must be set in the deployment workflow. No fallbacks allowed — fail fast if missing.
	if s := os.Getenv("OFFICIAL_SITE"); s != "" {
		return s
	}
	fmt.Fprintln(os.Stderr, "ERROR: OFFICIAL_SITE environment variable is required. Set it in your deploy workflow job as OFFICIAL_SITE: 'https://example.com'")
	os.Exit(1)
	return ""
}

// writeNavbar writes a simple _navbar.md that includes a link to the official site.
func writeNavbar(dir, siteURL string) error {
	var content string
	if siteURL != "" {
		content = fmt.Sprintf("[Home](README.md) • [Official Website](%s)\n", siteURL)
	} else {
		content = "[Home](README.md)\n"
	}
	path := filepath.Join(dir, "_navbar.md")
	return os.WriteFile(path, []byte(content), 0644)
}

// writeThemeCSS writes a CSS file to use a light content background with a deep-navy sidebar.
func writeThemeCSS(dir string) error {
	css := `/* Deep navy sidebar with light content theme for Docsify */
:root {
  --theme-color: #05386b;
  --text-color: #0f1720;
  --sidebar-bg: #022235;
  --content-bg: #f8fafc;
  --content-card-bg: #ffffff;
  --link-color: #0b69ff;
}
body {
  background: var(--content-bg);
  color: var(--text-color);
}
.sidebar {
  background: var(--sidebar-bg) !important;
  color: var(--text-color) !important;
}
.sidebar .sidebar-nav a {
  color: #e6eef8 !important;
}
.navbar, .app-name, .toolbar {
  background: linear-gradient(180deg, #032a45, #012034);
  color: #e6eef8 !important;
}
a {
  color: var(--link-color);
}
.markdown-section {
  background: var(--content-card-bg) !important;
  color: var(--text-color) !important;
  box-shadow: 0 1px 3px rgba(16,24,40,0.05);
  border-radius: 6px;
  padding: 1rem;
}
pre, code {
  background: #0f1720;
  color: #f1f5f9;
  border-radius: 4px;
  padding: 0.4rem;
}
`
	path := filepath.Join(dir, "docs.css")
	if err := os.WriteFile(path, []byte(css), 0644); err != nil {
		return err
	}
	return nil
}

// copyLogo copies logo/logo.webp to the docs directory as logo.webp if present in the repo
func copyLogo(dir string) error {
	src := filepath.Join("logo", "logo.webp")
	if _, err := os.Stat(src); os.IsNotExist(err) {
		// no logo provided
		return nil
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	dst := filepath.Join(dir, "logo.webp")
	return os.WriteFile(dst, data, 0644)
}

func titleFromFile(path string) string {
	// try to find first heading in file
	f, err := os.Open(path)
	if err != nil {
		return nameToTitle(filepath.Base(path))
	}
	defer func() {
		if err := f.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "close file error: %v\n", err)
		}
	}()
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
