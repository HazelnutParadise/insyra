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
				mdPath := filepath.ToSlash(f)
				if _, err := fmt.Fprintf(w, "* [%s](%s)\n", title, mdPath); err != nil {
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
			rel := filepath.ToSlash(f)
			title := titleFromFile(filepath.Join(*dirFlag, f))
			if _, err := fmt.Fprintf(w, "  * [%s](%s)\n", title, rel); err != nil {
				fmt.Fprintf(os.Stderr, "write error: %v\n", err)
				os.Exit(1)
			}
		}
	}

	fmt.Fprintf(os.Stderr, "generated %s\n", *outputFlag)

	// Also generate docs/index.html (Docsify entry) and navbar/theme files.
	tutorialSlugs := collectTutorialSlugs(entries)
	if err := writeIndex(*dirFlag, *repoFlag, tutorialSlugs); err != nil {
		fmt.Fprintf(os.Stderr, "generate index error: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "generated %s\n", filepath.Join(*dirFlag, "index.html"))

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

func writeIndex(dir, repoFlag string, tutorialSlugs []string) error {
	// Prefer explicit repo flag, otherwise use GitHub Actions env var
	repoURL := ""
	if repoFlag != "" {
		repoURL = repoFlag
	} else if repo := os.Getenv("GITHUB_REPOSITORY"); repo != "" {
		repoURL = "https://github.com/" + repo
	}
	slugSetJS := toJSStringArray(tutorialSlugs)

	tmpl := `<!DOCTYPE html>
<html>
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
	<link rel="icon" type="image/x-icon" href="https://insyra.hazelnut-paradise.com/favicon">
    <title>Insyra Docs</title>
    <meta name="description" content="Insyra documentation powered by Docsify" />
    <link rel="stylesheet" href="//unpkg.com/docsify/lib/themes/vue.css" />
    <!-- Custom theme overrides -->
    <link rel="stylesheet" href="docs.css" />
  </head>
  <body>
    <div id="app"></div>

    <script>
      const tutorialSlugs = new Set(%s);

      function splitRouteAndQuery(route) {
        const queryIndex = route.indexOf('?');
        if (queryIndex === -1) {
          return { path: route, query: '' };
        }
        return {
          path: route.slice(0, queryIndex),
          query: route.slice(queryIndex)
        };
      }

      function normalizeRoutePath(routePath) {
        const safePath = routePath || '/';
        const parts = safePath.split('/');
        const normalized = [];

        parts.forEach((part) => {
          if (!part || part === '.') {
            return;
          }
          if (part === '..') {
            normalized.pop();
            return;
          }
          normalized.push(part);
        });

        return '/' + normalized.join('/');
      }

      function stripMarkdownExtension(routePath) {
        const safePath = routePath || '/';
        const parts = safePath.split('/');
        const lastIndex = parts.length - 1;
        const lastPart = parts[lastIndex];

        if (lastPart && /\.md$/i.test(lastPart)) {
          parts[lastIndex] = lastPart.replace(/\.md$/i, '');
        }

        return parts.join('/') || '/';
      }

      function normalizeHashRoute(hash) {
        const safeHash = typeof hash === 'string' && hash.length > 0 ? hash : '#/';
        if (!safeHash.startsWith('#')) {
          return safeHash;
        }

        const route = safeHash.slice(1);
        const parsed = splitRouteAndQuery(route || '/');
        let normalizedPath = stripMarkdownExtension(normalizeRoutePath(parsed.path));

        const routeParts = normalizedPath.split('/').filter(Boolean);
        if (routeParts.length === 1) {
          const maybeTutorialSlug = routeParts[0];
          if (tutorialSlugs.has(maybeTutorialSlug)) {
            normalizedPath = '/tutorials/' + maybeTutorialSlug;
          }
        }

        return '#' + normalizedPath + parsed.query;
      }

      function getCanonicalRoutePath(hash) {
        const canonicalHash = normalizeHashRoute(hash || window.location.hash || '#/');
        const parsed = splitRouteAndQuery(canonicalHash.slice(1));
        return parsed.path || '/';
      }

      function canonicalizeCurrentHash() {
        const currentHash = window.location.hash || '#/';
        const canonicalHash = normalizeHashRoute(currentHash);
        if (currentHash !== canonicalHash) {
          window.location.replace(canonicalHash);
          return true;
        }
        return false;
      }

      function normalizeRenderedContentLinks() {
        const currentRoutePath = getCanonicalRoutePath(window.location.hash || '#/');
        const currentDir = currentRoutePath.endsWith('/')
          ? currentRoutePath
          : currentRoutePath.slice(0, currentRoutePath.lastIndexOf('/') + 1);

        const contentLinks = document.querySelectorAll('.markdown-section a[href]');
        contentLinks.forEach((link) => {
          const href = link.getAttribute('href');
          if (!href || href.startsWith('http://') || href.startsWith('https://') || href.startsWith('mailto:')) {
            return;
          }
          if (href.startsWith('#') && !href.startsWith('#/')) {
            return;
          }

          let normalizedHref = href;

          if (href.startsWith('#/')) {
            normalizedHref = normalizeHashRoute(href);
          } else if ((href.startsWith('./') || href.startsWith('../') || href.startsWith('/')) && /\.md($|\?)/i.test(href)) {
            const parsed = splitRouteAndQuery(href);
            let targetPath = '';
            if (parsed.path.startsWith('/')) {
              targetPath = parsed.path;
            } else {
              targetPath = currentDir + parsed.path;
            }
            targetPath = stripMarkdownExtension(normalizeRoutePath(targetPath));

            if (currentRoutePath === '/tutorials/README') {
              const routeParts = targetPath.split('/').filter(Boolean);
              if (routeParts.length === 1 && tutorialSlugs.has(routeParts[0])) {
                targetPath = '/tutorials/' + routeParts[0];
              }
            }

            normalizedHref = '#' + targetPath + parsed.query;
          }

          if (normalizedHref !== href) {
            link.setAttribute('href', normalizedHref);
          }
        });
      }

      canonicalizeCurrentHash();

      window.$docsify = {
        name: 'Insyra',
        repo: '%s',
        loadSidebar: true,
        loadNavbar: true,
        logo: 'logo.webp',
        relativePath: false,
        alias: {
          '/.*/_sidebar.md': '/_sidebar.md',
          '/.*/_navbar.md': '/_navbar.md'
        },
        plugins: [
          function dynamicRoutePlugin(hook) {
            hook.init(function () {
              canonicalizeCurrentHash();
            });
            hook.ready(function () {
              if (canonicalizeCurrentHash()) {
                return;
              }
              normalizeRenderedContentLinks();
            });
            hook.doneEach(function () {
              if (canonicalizeCurrentHash()) {
                return;
              }
              normalizeRenderedContentLinks();
            });
          }
        ]
      };
    </script>

    <script src="//unpkg.com/docsify/lib/docsify.min.js"></script>
	<script src="//unpkg.com/docsify-copy-code"></script>
	<script src="//cdn.jsdelivr.net/npm/prismjs@1.22/components/prism-go.min.js"></script>
  </body>
</html>
`

	content := fmt.Sprintf(tmpl, slugSetJS, repoURL)
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
		content = fmt.Sprintf("[Home](README.md) | [Official Website](%s)\n", siteURL)
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
  color: #d8e7f8 !important;
}
.sidebar .sidebar-nav a {
  color: #e6eef8 !important;
}
.sidebar .sidebar-nav {
  padding-bottom: 1rem;
}
.sidebar .sidebar-nav ul {
  margin: 0.25rem 0 0.85rem;
  padding-left: 1rem;
}
.sidebar .sidebar-nav li {
  margin: 0.15rem 0;
}
.sidebar .sidebar-nav a {
  display: block;
  border-radius: 0.4rem;
  padding: 0.32rem 0.5rem;
  transition: background-color 0.16s ease, color 0.16s ease;
  line-height: 1.35;
}
.sidebar .sidebar-nav a:hover {
  background: rgba(149, 188, 238, 0.17);
  color: #ffffff !important;
}
.sidebar .sidebar-nav a.active {
  background: rgba(149, 188, 238, 0.3);
  color: #ffffff !important;
  font-weight: 700;
}
.sidebar .sidebar-nav p {
  color: #b7d4f0 !important;
  margin: 0.8rem 0 0.4rem;
  padding: 0.1rem 0.5rem;
  font-size: 0.88rem;
  text-transform: uppercase;
  letter-spacing: 0.06em;
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

func collectTutorialSlugs(entries map[string][]string) []string {
	files, ok := entries["tutorials"]
	if !ok {
		return nil
	}
	slugs := make([]string, 0, len(files))
	for _, rel := range files {
		base := filepath.Base(rel)
		if strings.EqualFold(base, "README.md") {
			continue
		}
		slug := strings.TrimSuffix(base, filepath.Ext(base))
		if slug != "" {
			slugs = append(slugs, slug)
		}
	}
	sort.Strings(slugs)
	return slugs
}

func toJSStringArray(values []string) string {
	if len(values) == 0 {
		return "[]"
	}
	quoted := make([]string, 0, len(values))
	for _, v := range values {
		quoted = append(quoted, fmt.Sprintf("%q", v))
	}
	return "[" + strings.Join(quoted, ", ") + "]"
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
