// Minimal, dependency-free markdown parser that produces a token tree —
// never an HTML string. The `text` display block's payload is user/LLM
// generated prose, so BlockText.svelte renders these tokens through plain
// Svelte text interpolation (which auto-escapes) and never {@html}; there is
// no raw-HTML injection surface because no HTML string is ever produced
// here, sanitized or not.
//
// Supported subset: #..###### headings, blank-line paragraphs, - / * / 1.
// lists, ``` fenced code, > blockquotes, and inline **bold** / *italic* /
// `code` / [text](href) with href restricted to http(s)/mailto/relative —
// anything else (javascript:, data:, etc.) degrades to plain text so a
// malicious link can never become a navigable href.

export type InlineToken =
  | { kind: "text"; text: string }
  | { kind: "bold"; text: string }
  | { kind: "italic"; text: string }
  | { kind: "code"; text: string }
  | { kind: "link"; text: string; href: string };

export type MarkdownBlock =
  | { kind: "heading"; level: number; inline: InlineToken[] }
  | { kind: "paragraph"; inline: InlineToken[] }
  | { kind: "list"; ordered: boolean; items: InlineToken[][] }
  | { kind: "code"; content: string }
  | { kind: "quote"; inline: InlineToken[] };

const SAFE_HREF_RE = /^(https?:|mailto:|\/|#)/i;

const INLINE_TOKEN_RE =
  /\*\*(.+?)\*\*|\*(.+?)\*|`(.+?)`|\[([^\]]*)\]\(([^)]+)\)/g;

export function parseInline(text: string): InlineToken[] {
  const tokens: InlineToken[] = [];
  let lastIndex = 0;

  INLINE_TOKEN_RE.lastIndex = 0;
  let match: RegExpExecArray | null;

  // eslint-disable-next-line no-cond-assign -- standard exec-loop scan shape
  while ((match = INLINE_TOKEN_RE.exec(text)) !== null) {
    if (match.index > lastIndex) {
      tokens.push({ kind: "text", text: text.slice(lastIndex, match.index) });
    }

    const [, bold, italic, code, linkText, linkHref] = match;
    if (bold !== undefined) {
      tokens.push({ kind: "bold", text: bold });
    } else if (italic !== undefined) {
      tokens.push({ kind: "italic", text: italic });
    } else if (code !== undefined) {
      tokens.push({ kind: "code", text: code });
    } else if (linkText !== undefined && linkHref !== undefined) {
      if (SAFE_HREF_RE.test(linkHref.trim())) {
        tokens.push({ kind: "link", text: linkText, href: linkHref.trim() });
      } else {
        // Unsafe scheme (javascript:, data:, vbscript:, ...) — render as
        // plain visible text instead of dropping the content silently.
        tokens.push({ kind: "text", text: `${linkText} (${linkHref})` });
      }
    }

    lastIndex = match.index + match[0].length;
  }

  if (lastIndex < text.length) {
    tokens.push({ kind: "text", text: text.slice(lastIndex) });
  }

  return tokens;
}

const HEADING_RE = /^(#{1,6})\s+(.*)$/;
const UNORDERED_ITEM_RE = /^[-*]\s+(.*)$/;
const ORDERED_ITEM_RE = /^\d+\.\s+(.*)$/;
const QUOTE_RE = /^>\s?(.*)$/;
const FENCE_RE = /^```/;

export function parseMarkdown(source: string): MarkdownBlock[] {
  const lines = source.split("\n");
  const blocks: MarkdownBlock[] = [];

  let i = 0;
  let paragraphLines: string[] = [];

  const flushParagraph = (): void => {
    if (paragraphLines.length === 0) {
      return;
    }

    blocks.push({
      kind: "paragraph",
      inline: parseInline(paragraphLines.join(" ")),
    });
    paragraphLines = [];
  };

  while (i < lines.length) {
    const line = lines[i] ?? "";

    if (FENCE_RE.test(line)) {
      flushParagraph();
      const codeLines: string[] = [];
      i++;
      while (i < lines.length && !FENCE_RE.test(lines[i] ?? "")) {
        codeLines.push(lines[i] ?? "");
        i++;
      }
      i++; // consume closing fence
      blocks.push({ kind: "code", content: codeLines.join("\n") });
      continue;
    }

    const heading = HEADING_RE.exec(line);
    if (heading) {
      flushParagraph();
      blocks.push({
        kind: "heading",
        level: (heading[1] ?? "#").length,
        inline: parseInline(heading[2] ?? ""),
      });
      i++;
      continue;
    }

    const quote = QUOTE_RE.exec(line);
    if (quote) {
      flushParagraph();
      blocks.push({ kind: "quote", inline: parseInline(quote[1] ?? "") });
      i++;
      continue;
    }

    const unordered = UNORDERED_ITEM_RE.exec(line);
    const ordered = ORDERED_ITEM_RE.exec(line);
    if (unordered || ordered) {
      flushParagraph();
      const isOrdered = ordered !== null;
      const items: InlineToken[][] = [];

      while (i < lines.length) {
        const itemLine = lines[i] ?? "";
        const itemMatch = isOrdered
          ? ORDERED_ITEM_RE.exec(itemLine)
          : UNORDERED_ITEM_RE.exec(itemLine);
        if (!itemMatch) {
          break;
        }
        items.push(parseInline(itemMatch[1] ?? ""));
        i++;
      }

      blocks.push({ kind: "list", ordered: isOrdered, items });
      continue;
    }

    if (line.trim() === "") {
      flushParagraph();
      i++;
      continue;
    }

    paragraphLines.push(line);
    i++;
  }

  flushParagraph();

  return blocks;
}
