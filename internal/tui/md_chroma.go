package tui

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
	extast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	gmtext "github.com/yuin/goldmark/text"
)

// issueRefRE matches owner/repo#NNN (aligned with openclaude3 markdown linkify).
var issueRefRE = regexp.MustCompile(`(^|[^\w./-])([A-Za-z0-9][\w-]*/[A-Za-z0-9][\w.-]*)#(\d+)\b`)

var assistantMDParser = goldmark.New(
	goldmark.WithExtensions(
		extension.Table,
		extension.Linkify,
		extension.TaskList,
	),
)

type mdChromaRenderer struct {
	src   []byte
	width int
	dark  bool
}

func newMDChromaRenderer(src []byte, width int, dark bool) *mdChromaRenderer {
	return &mdChromaRenderer{src: src, width: width, dark: dark}
}

func (r *mdChromaRenderer) chromaStyle() *chroma.Style {
	if r.dark {
		return styles.Get("monokai")
	}
	return styles.Get("github")
}

func (r *mdChromaRenderer) highlightCode(code, lang string) string {
	code = strings.TrimSuffix(code, "\n")
	if code == "" {
		return ""
	}
	var lexer chroma.Lexer
	if strings.TrimSpace(lang) != "" {
		lexer = lexers.Get(strings.TrimSpace(lang))
	}
	if lexer == nil {
		lexer = lexers.Analyse(code)
	}
	if lexer == nil {
		lexer = lexers.Fallback
	}
	it, err := lexer.Tokenise(nil, code)
	if err != nil {
		return code + "\n"
	}
	formatter := formatters.Get("terminal16m")
	if formatter == nil {
		formatter = formatters.Get("terminal256")
	}
	if formatter == nil {
		formatter = formatters.Fallback
	}
	var buf bytes.Buffer
	if err := formatter.Format(&buf, r.chromaStyle(), it); err != nil {
		return code + "\n"
	}
	return buf.String()
}

func (r *mdChromaRenderer) wrapBlock(s string) string {
	s = strings.TrimRight(s, "\n")
	if r.width <= 0 {
		return s
	}
	lines := strings.Split(s, "\n")
	var out []string
	for _, ln := range lines {
		if lipgloss.Width(ln) <= r.width {
			out = append(out, ln)
			continue
		}
		out = append(out, strings.Split(ansi.Wordwrap(ln, r.width, ""), "\n")...)
	}
	return strings.Join(out, "\n")
}

func linkifyIssueRefs(s string) string {
	return issueRefRE.ReplaceAllStringFunc(s, func(m string) string {
		sm := issueRefRE.FindStringSubmatch(m)
		if len(sm) != 4 {
			return m
		}
		prefix, repo, num := sm[1], sm[2], sm[3]
		url := fmt.Sprintf("https://github.com/%s/issues/%s", repo, num)
		label := repo + "#" + num
		return prefix + termenv.Hyperlink(url, label)
	})
}

func (r *mdChromaRenderer) renderDocument(doc *ast.Document) string {
	var b strings.Builder
	for n := doc.FirstChild(); n != nil; n = n.NextSibling() {
		b.WriteString(r.renderBlock(n))
	}
	return strings.TrimRight(b.String(), "\n")
}

func (r *mdChromaRenderer) renderBlock(n ast.Node) string {
	switch n := n.(type) {
	case *ast.Heading:
		return r.renderHeading(n)
	case *ast.Paragraph:
		return r.wrapBlock(r.renderParagraphBlock(n)) + "\n"
	case *ast.FencedCodeBlock:
		return r.renderFencedCode(n) + "\n"
	case *ast.CodeBlock:
		return r.highlightCode(string(n.Lines().Value(r.src)), "") + "\n"
	case *ast.Blockquote:
		return r.renderBlockquote(n)
	case *ast.List:
		return r.renderList(n, 0)
	case *ast.ThematicBreak:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("---") + "\n"
	case *extast.Table:
		return r.renderTable(n) + "\n"
	case *ast.HTMLBlock:
		return ""
	default:
		return ""
	}
}

func (r *mdChromaRenderer) renderHeading(h *ast.Heading) string {
	text := strings.TrimRight(r.renderInlines(h, r.src, false), "\n")
	var st lipgloss.Style
	switch h.Level {
	case 1:
		st = lipgloss.NewStyle().Bold(true).Italic(true).Underline(true)
	case 2:
		st = lipgloss.NewStyle().Bold(true)
	default:
		st = lipgloss.NewStyle().Bold(true)
	}
	return r.wrapBlock(st.Render(text)) + "\n\n"
}

func (r *mdChromaRenderer) renderParagraphBlock(p *ast.Paragraph) string {
	return strings.TrimRight(r.renderInlines(p, r.src, false), "\n")
}

func (r *mdChromaRenderer) renderFencedCode(f *ast.FencedCodeBlock) string {
	lang := string(f.Language(r.src))
	body := string(f.Lines().Value(r.src))
	return r.highlightCode(body, lang)
}

func (r *mdChromaRenderer) renderBlockquote(bq *ast.Blockquote) string {
	var ib strings.Builder
	for c := bq.FirstChild(); c != nil; c = c.NextSibling() {
		ib.WriteString(r.renderBlock(c))
	}
	inner := strings.TrimRight(ib.String(), "\n")
	if inner == "" {
		return ""
	}
	bar := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("│")
	it := lipgloss.NewStyle().Italic(true)
	var lines []string
	for _, ln := range strings.Split(inner, "\n") {
		plain := strings.TrimSpace(ansi.Strip(ln))
		if plain == "" {
			lines = append(lines, "")
			continue
		}
		lines = append(lines, bar+" "+it.Render(ln))
	}
	return strings.Join(lines, "\n") + "\n"
}

func (r *mdChromaRenderer) renderList(list *ast.List, depth int) string {
	var b strings.Builder
	idx := 0
	for item := list.FirstChild(); item != nil; item = item.NextSibling() {
		li, ok := item.(*ast.ListItem)
		if !ok {
			continue
		}
		idx++
		b.WriteString(r.renderListItem(li, list, depth, idx))
	}
	return b.String()
}

func (r *mdChromaRenderer) listMarker(depth, n int) string {
	switch depth {
	case 0, 1:
		return fmt.Sprintf("%d.", n)
	case 2:
		return fmt.Sprintf("%c.", 'a'+rune((n-1)%26))
	case 3:
		return strings.ToLower(toRoman(n)) + "."
	default:
		return fmt.Sprintf("%d.", n)
	}
}

func (r *mdChromaRenderer) renderListItem(li *ast.ListItem, parent *ast.List, depth, index int) string {
	indent := strings.Repeat("  ", depth)
	var marker string
	if parent.IsOrdered() {
		marker = r.listMarker(depth, parent.Start+index-1)
	} else {
		marker = "-"
	}
	var taskExtra string
	var blocks []ast.Node
	for c := li.FirstChild(); c != nil; c = c.NextSibling() {
		if tb, ok := c.(*extast.TaskCheckBox); ok {
			if tb.IsChecked {
				taskExtra = "[x] "
			} else {
				taskExtra = "[ ] "
			}
			continue
		}
		blocks = append(blocks, c)
	}
	lineStart := indent + marker + " " + taskExtra
	contPad := indent + strings.Repeat(" ", lipgloss.Width(marker)+1)

	var lines []string
	first := true
	for _, c := range blocks {
		switch n := c.(type) {
		case *ast.List:
			lines = append(lines, strings.TrimRight(r.renderList(n, depth+1), "\n"))
			first = false
		case *ast.Paragraph:
			body := r.wrapBlock(r.renderParagraphBlock(n))
			if first {
				lines = append(lines, lineStart+body)
				first = false
			} else {
				for _, ln := range strings.Split(body, "\n") {
					lines = append(lines, contPad+ln)
				}
			}
		default:
			blk := strings.TrimRight(r.renderBlock(c), "\n")
			if first {
				lines = append(lines, lineStart+blk)
				first = false
			} else {
				for _, ln := range strings.Split(blk, "\n") {
					lines = append(lines, contPad+ln)
				}
			}
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func (r *mdChromaRenderer) renderTable(t *extast.Table) string {
	var headers []*extast.TableCell
	var rows [][]*extast.TableCell
	for c := t.FirstChild(); c != nil; c = c.NextSibling() {
		switch n := c.(type) {
		case *extast.TableHeader:
			for cell := n.FirstChild(); cell != nil; cell = cell.NextSibling() {
				if tc, ok := cell.(*extast.TableCell); ok {
					headers = append(headers, tc)
				}
			}
		case *extast.TableRow:
			var row []*extast.TableCell
			for cell := n.FirstChild(); cell != nil; cell = cell.NextSibling() {
				if tc, ok := cell.(*extast.TableCell); ok {
					row = append(row, tc)
				}
			}
			rows = append(rows, row)
		}
	}
	if len(headers) == 0 {
		return ""
	}
	nCol := len(headers)
	cellStr := func(tc *extast.TableCell) string {
		return strings.TrimSpace(r.renderTableCell(tc))
	}
	widths := make([]int, nCol)
	colAlign := make([]extast.Alignment, nCol)
	for i, h := range headers {
		widths[i] = max(3, lipgloss.Width(cellStr(h)))
		if i < len(t.Alignments) {
			colAlign[i] = t.Alignments[i]
		}
	}
	for _, row := range rows {
		for i := 0; i < nCol && i < len(row); i++ {
			w := lipgloss.Width(cellStr(row[i]))
			if w > widths[i] {
				widths[i] = w
			}
		}
	}
	var b strings.Builder
	writeRow := func(cells []*extast.TableCell, bold bool) {
		b.WriteString("| ")
		for i := 0; i < nCol; i++ {
			var s string
			var a extast.Alignment
			if i < len(cells) && cells[i] != nil {
				s = cellStr(cells[i])
				a = cells[i].Alignment
			}
			if a == extast.AlignNone && i < len(colAlign) {
				a = colAlign[i]
			}
			pad := padTableCell(s, lipgloss.Width(s), widths[i], a)
			if bold {
				pad = lipgloss.NewStyle().Bold(true).Render(pad)
			}
			b.WriteString(pad)
			b.WriteString(" | ")
		}
		b.WriteByte('\n')
	}
	writeRow(headers, true)
	b.WriteString("|")
	for i := 0; i < nCol; i++ {
		b.WriteString(strings.Repeat("-", widths[i]+2))
		b.WriteByte('|')
	}
	b.WriteByte('\n')
	for _, row := range rows {
		cells := row
		for len(cells) < nCol {
			cells = append(cells, nil)
		}
		writeRow(cells, false)
	}
	return strings.TrimRight(b.String(), "\n")
}

func (r *mdChromaRenderer) renderTableCell(tc *extast.TableCell) string {
	if tc.Lines().Len() == 0 {
		return ""
	}
	raw := strings.TrimSpace(string(tc.Lines().Value(r.src)))
	if raw == "" {
		return ""
	}
	// GFM stores cell text in Lines(), not as child AST; re-parse for inline emphasis/links.
	sub := assistantMDParser.Parser().Parse(gmtext.NewReader([]byte(raw)))
	d, ok := sub.(*ast.Document)
	if !ok {
		return linkifyIssueRefs(raw)
	}
	subSrc := []byte(raw)
	prev := r.src
	r.src = subSrc
	defer func() { r.src = prev }()

	var parts []string
	for c := d.FirstChild(); c != nil; c = c.NextSibling() {
		switch n := c.(type) {
		case *ast.Paragraph:
			parts = append(parts, strings.TrimSpace(r.renderParagraphBlock(n)))
		default:
			parts = append(parts, strings.TrimSpace(strings.TrimRight(r.renderBlock(c), "\n")))
		}
	}
	if len(parts) == 0 {
		return linkifyIssueRefs(raw)
	}
	return strings.Join(parts, " ")
}

func padTableCell(content string, displayW, targetW int, a extast.Alignment) string {
	pad := max(0, targetW-displayW)
	switch a {
	case extast.AlignCenter:
		left := pad / 2
		return strings.Repeat(" ", left) + content + strings.Repeat(" ", pad-left)
	case extast.AlignRight:
		return strings.Repeat(" ", pad) + content
	default:
		return content + strings.Repeat(" ", pad)
	}
}

func (r *mdChromaRenderer) renderInlines(parent ast.Node, source []byte, insideLink bool) string {
	var b strings.Builder
	for n := parent.FirstChild(); n != nil; n = n.NextSibling() {
		b.WriteString(r.renderInline(n, source, insideLink))
	}
	return b.String()
}

// insideLink avoids nested OSC8 when walking link label text (openclaude3 behavior).
func (r *mdChromaRenderer) renderInline(n ast.Node, source []byte, insideLink bool) string {
	switch n := n.(type) {
	case *ast.Text:
		s := string(n.Value(source))
		if n.SoftLineBreak() {
			if insideLink {
				return s + "\n"
			}
			return s + " "
		}
		if n.HardLineBreak() {
			return s + "\n"
		}
		if insideLink {
			return s
		}
		return linkifyIssueRefs(s)
	case *ast.String:
		return string(n.Value)
	case *ast.CodeSpan:
		var buf strings.Builder
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			if t, ok := c.(*ast.Text); ok {
				buf.Write(t.Value(source))
			}
		}
		return lipgloss.NewStyle().Foreground(lipgloss.Color("141")).Render(buf.String())
	case *ast.Emphasis:
		inner := r.renderInlines(n, source, insideLink)
		switch {
		case n.Level >= 3:
			return lipgloss.NewStyle().Bold(true).Italic(true).Render(inner)
		case n.Level == 2:
			return lipgloss.NewStyle().Bold(true).Render(inner)
		default:
			return lipgloss.NewStyle().Italic(true).Render(inner)
		}
	case *ast.Link:
		label := r.renderInlines(n, source, true)
		href := string(n.Destination)
		if strings.HasPrefix(strings.ToLower(href), "mailto:") {
			return strings.TrimPrefix(href, "mailto:")
		}
		plain := ansi.Strip(label)
		if plain != "" && plain != href {
			return termenv.Hyperlink(href, label)
		}
		return termenv.Hyperlink(href, href)
	case *ast.Image:
		return string(n.Destination)
	case *ast.AutoLink:
		u := string(n.URL(source))
		lbl := string(n.Label(source))
		if n.AutoLinkType == ast.AutoLinkEmail {
			return lbl
		}
		return termenv.Hyperlink(u, u)
	case *extast.TaskCheckBox:
		if n.IsChecked {
			return "[x]"
		}
		return "[ ]"
	default:
		return ""
	}
}

func toRoman(n int) string {
	if n <= 0 {
		return "i"
	}
	vals := []struct {
		v int
		s string
	}{
		{1000, "M"}, {900, "CM"}, {500, "D"}, {400, "CD"},
		{100, "C"}, {90, "XC"}, {50, "L"}, {40, "XL"},
		{10, "X"}, {9, "IX"}, {5, "V"}, {4, "IV"}, {1, "I"},
	}
	var b strings.Builder
	x := n
	for _, p := range vals {
		for x >= p.v {
			b.WriteString(p.s)
			x -= p.v
		}
	}
	return b.String()
}

// renderAssistantMarkdownChroma renders assistant markdown using goldmark + Chroma (v3-style richness).
// When trimEdges is false (streaming), leading/trailing space is preserved except an all-whitespace buffer renders as empty.
func renderAssistantMarkdownChroma(width int, md string, dark, trimEdges bool) string {
	if trimEdges {
		md = strings.TrimSpace(md)
	} else if strings.TrimSpace(md) == "" {
		return ""
	}
	if md == "" {
		return ""
	}
	w := width
	if w < 40 {
		w = 40
	}
	if w > 120 {
		w = 120
	}
	src := []byte(md)
	doc := assistantMDParser.Parser().Parse(gmtext.NewReader(src))
	d, ok := doc.(*ast.Document)
	if !ok {
		return ""
	}
	rend := newMDChromaRenderer(src, w, dark)
	out := rend.renderDocument(d)
	return strings.TrimRight(out, "\n")
}
