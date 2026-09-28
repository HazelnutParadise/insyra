# Design

## Context

`DecodingReader` normalises a name (lower case, separators removed) and looks it up in `decoders`; a miss is the unsupported-encoding error. See proposal.md for why the misses listed there should read.

## Goals / Non-Goals

**Goals:** the names v0.3.2 read through its substring rules, and the standard labels of the same charsets, read again.

**Non-Goals:** any fallback for a name outside the table. The refusal of an unknown name stays exactly as it is.

## Decisions

- **Explicit table entries, not a label index.** `golang.org/x/text/encoding/htmlindex` knows the WHATWG labels, including every Big5 and GBK alias here, and could be consulted after a table miss. It is not used, because it also maps names to encodings this line deliberately refuses: `iso-2022-kr`, `iso-2022-cn` and `hz-gb-2312` resolve to WHATWG's replacement encoding, which decodes every byte to U+FFFD, and `ascii` or `latin1` would resolve to Windows-1252 if the table ever lost them. It also does not know `utf-8-sig`. A fixed list keeps the accepted set visible in the table and in the error message.
- **`utf-8-sig` strips the mark.** It maps to `unicode.UTF8BOM`, whose decoder removes a leading byte-order mark, which is what the name means where it comes from (Python's codec). Plain `utf-8` keeps passing bytes through unchanged, as it does today.
- **GBK names decode through GB18030**, like `gbk` and `gb2312` already do: GB18030 is a superset, so a GBK or GB2312 file decodes the same.

## Risks / Trade-offs

- [A future alias is still refused] → the unsupported-encoding error lists every accepted name, so the caller can pick one.
