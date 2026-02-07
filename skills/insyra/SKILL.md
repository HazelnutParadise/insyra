---
name: insyra
description: Use when working in Go and needing DataList/DataTable-style data wrangling, quick previews, parallel transforms, file conversion (CSV/Excel/Parquet), Excel-like column formulas (CCL), or charts—especially when turning messy external data into structured tables for downstream processing, automation, reporting, or MCP tools.
---

# Insyra (Go)

## Overview
**Insyra** is a Go library for dataframe-like workflows: ingest → clean/transform → summarize → visualize/export. It’s useful even when the end goal isn’t “data analysis” (e.g., automation, scraping, QA, reporting).

## When to Use (not only analytics)
Use Insyra when you need any of these in Go:

- **ETL / data cleaning:** normalize columns, derive new columns, filter/sort/group-like operations.
- **File chores:** CSV/Excel conversion, Parquet read/write, lightweight data interchange.
- **External data → structure:** wrap crawler/API results into a `DataTable` to make downstream steps simpler.
- **Quick inspection / debugging:** fast console preview for “what’s inside this thing?”
- **Parallel data transforms:** speed up map/filter style workloads safely.
- **Generate artifacts:** charts (static or interactive) and export outputs for docs/reports.
- **Agent tools / MCP servers:** implement tools that accept JSON inputs, run tabular transforms, and return computed summaries.

## Core mental model
- **`DataList`**: a column/series-like container (stats, sort, transform).
- **`DataTable`**: multiple named `DataList` columns as a table.
- **`isr` syntactic sugar**: preferred entrypoint for new codebases (clearer, shorter).
- **CCL (Column Calculation Language)**: Excel-like formulas for derived columns.
- **Instance error tracking**: chain fluent ops, then check `Err()` / `ClearErr()`.
- **Concurrency safety**: use the project’s recommended atomic/concurrent helpers when doing multi-step concurrent operations.

## Quick reference (what to look up)
When you’re unsure, go straight to the upstream docs for the latest API:

### Insyra upstream docs
- Official docs site: https://hazelnutparadise.github.io/insyra/
- Go package docs: https://pkg.go.dev/github.com/HazelnutParadise/insyra
- Docs folder (often newest): https://github.com/HazelnutParadise/insyra/tree/main/Docs
- Releases (API vs version): https://github.com/HazelnutParadise/insyra/releases

### Insyra docs via MCP (recommended for agents)
If you want **up-to-date Insyra documentation inside an MCP-capable client**, prefer these:

- Insyra docs MCP server (GitMCP): https://gitmcp.io/HazelnutParadise/insyra
- Context7 (docs MCP server / alternative):
  - https://context7.com/
  - https://github.com/upstash/context7

### MCP implementation references (optional)
Only needed if you’re **building your own MCP server/tools** around Insyra:

- Official Go SDK package docs: https://pkg.go.dev/github.com/modelcontextprotocol/go-sdk/mcp
- Official Go SDK repo: https://github.com/modelcontextprotocol/go-sdk

## Typical agent workflow (high-level)
1. **Clarify the artifact**: table output? chart? exported file? computed metrics? tool response schema?
2. **Choose the minimal Insyra structures**: `DataList` for single-series; `DataTable` for tabular.
3. **Prefer readable transforms**:
   - Start with `isr` sugar when writing new snippets.
   - Use CCL if the transform is naturally “formula-like”.
4. **Export / return**:
   - File outputs (CSV/Excel/Parquet/chart) when humans need artifacts.
   - JSON-friendly summaries when implementing MCP tools.
5. **Version sanity**: if docs mention a method that “doesn’t exist”, check the corresponding **release tag** docs.

## Common mistakes
- **Using docs for `main` while pinned to an older release** → always cross-check `releases/`.
- **Calling many chained ops and forgetting `Err()`** → check error state before trusting results.
- **Using concurrent goroutines on the same instance without the recommended atomic helper** → follow upstream guidance for thread safety.
- **Treating Insyra as “only for analytics”** → it’s often the fastest way to make messy data *structured*, which simplifies everything else.
