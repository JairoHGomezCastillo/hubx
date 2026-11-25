package evidence

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Page geometry in PDF points (US Letter). The card follows the HUBX "airport
// apron" visual language of the MFE: a light apron page, a white surface card,
// an asphalt header band and yellow/blue accents.
const (
	pageWidth     = 612
	pageHeight    = 792
	cardX         = 36
	cardY         = 36
	cardWidth     = 540
	cardHeight    = 720
	headerY       = 704
	headerHeight  = 52
	contentTop    = 682
	contentBottom = 92
	textX         = 56
	canonicalX    = 72
	proseWidth    = 380
)

// Normalized RGB triples from hubx-mfe/src/index.css, written as PDF operators.
const (
	colorApron     = "0.910 0.929 0.933" // --apron #e8edee
	colorSurface   = "1 1 1"             // --surface #fff
	colorInk       = "0.106 0.145 0.157" // --ink #1b2528
	colorInkSoft   = "0.322 0.376 0.396" // --ink-soft #526065
	colorLine      = "0.776 0.816 0.824" // --line #c6d0d2
	colorAsphalt   = "0.188 0.224 0.235" // --asphalt #30393c
	colorOnAsphalt = "0.949 0.961 0.961" // --on-asphalt #f2f5f5
	colorYellow    = "0.969 0.741 0.086" // --yellow #f7bd16
	colorBlue      = "0.110 0.310 0.769" // --blue #1c4fc4
)

type lineStyle int

const (
	styleField lineStyle = iota
	styleSection
	styleUpdate
	styleCanonical
	styleFingerprint
	styleProse
)

// metrics returns the font resource, size, leading and color of a line.
func (s lineStyle) metrics() (font, size string, leading float64, color string) {
	switch s {
	case styleSection:
		return "/F2", "10", 22, colorBlue
	case styleUpdate:
		return "/F2", "11", 20, colorAsphalt
	case styleCanonical:
		return "/F1", "9", 12, colorInkSoft
	case styleFingerprint:
		return "/F2", "9", 15, colorInk
	case styleProse:
		return "/F1", "9", 13, colorInk
	default:
		return "/F1", "10", 15, colorInk
	}
}

func (s lineStyle) leadingHeight() float64 {
	_, _, leading, _ := s.metrics()
	return leading
}

type pdfLine struct {
	text  string
	style lineStyle
	block int // nonzero groups the seven canonical lines of one update
}

// Render writes a self-contained, uncompressed PDF document. Every document
// line is emitted as one literal PDF string so a recipient can read the text
// directly from the file; no font embedding or external resource is needed.
func Render(doc *Evidence) []byte {
	return writePDF(paginate(documentLines(doc)))
}

func documentLines(doc *Evidence) []pdfLine {
	lines := []pdfLine{{text: "Shipment", style: styleSection}}
	add := func(text string) {
		lines = append(lines, pdfLine{text: text, style: styleField})
	}
	add("Code: " + doc.Shipment.Code)
	add("Origin: " + doc.Shipment.Origin)
	add("Destination: " + doc.Shipment.Destination)
	add("Flight: " + doc.Shipment.Flight)
	add("Weight: " + doc.Shipment.WeightKg + " kg")
	add("Customer: " + doc.Shipment.CustomerName)
	add("Current stage: " + doc.Shipment.Stage)
	lines = append(lines, pdfLine{text: "Updates:", style: styleSection})
	if len(doc.Updates) == 0 {
		lines = append(lines, pdfLine{text: "The shipment has no updates yet.", style: styleProse})
	}
	for i, update := range doc.Updates {
		lines = append(lines, pdfLine{
			text:  fmt.Sprintf("%d. %s", update.Sequence, update.Stage),
			style: styleUpdate,
		})
		add("Responsible party: " + update.ResponsibleParty)
		if update.AssignedPartner != "" {
			add("Assigned partner: " + update.AssignedPartner)
		}
		add("Recorded at: " + update.RecordedAt.UTC().Format(time.RFC3339Nano))
		// The canonical seven lines are the exact HU-03 fingerprint input.
		// They stay consecutive, unwrapped and free of decoration strings.
		for _, line := range strings.Split(update.CanonicalInput, "\n") {
			lines = append(lines, pdfLine{text: line, style: styleCanonical, block: i + 1})
		}
		lines = append(lines,
			pdfLine{text: "Fingerprint: " + update.Fingerprint, style: styleFingerprint},
			// Deliberately not "Previous fingerprint": an external checker must
			// not mistake the predecessor link for the update's own fingerprint.
			pdfLine{text: "Previous: " + update.PreviousFingerprint, style: styleField},
		)
	}
	add("Generated at: " + doc.GeneratedAt.UTC().Format(time.RFC3339))
	lines = append(lines, pdfLine{text: "How to verify the fingerprints:", style: styleSection})
	for _, instruction := range strings.Split(doc.Instructions, "\n") {
		for _, fragment := range wrapLine(instruction, proseWidth, 9) {
			lines = append(lines, pdfLine{text: fragment, style: styleProse})
		}
	}
	// One sentence per line keeps both scope phrases intact after PDF parsing.
	for _, scope := range []string{
		"Scope: This evidence proves that the shipment history is consistent with itself.",
		"It does not by itself prove that HUBX did not rebuild the whole history.",
	} {
		for _, fragment := range wrapLine(scope, proseWidth, 9) {
			lines = append(lines, pdfLine{text: fragment, style: styleProse})
		}
	}
	return lines
}

// paginate groups canonical blocks together so a page break can never split
// the seven-line fingerprint input of an update.
func paginate(lines []pdfLine) [][]pdfLine {
	pages := make([][]pdfLine, 0, 2)
	current := make([]pdfLine, 0, len(lines))
	y := float64(contentTop)
	for i := 0; i < len(lines); {
		group := []pdfLine{lines[i]}
		if lines[i].block != 0 {
			for i+len(group) < len(lines) && lines[i+len(group)].block == lines[i].block {
				group = append(group, lines[i+len(group)])
			}
		}
		groupHeight := 0.0
		for _, line := range group {
			groupHeight += line.style.leadingHeight()
		}
		if len(current) > 0 && y-groupHeight < contentBottom {
			pages = append(pages, current)
			current = make([]pdfLine, 0, len(lines))
			y = contentTop
		}
		for _, line := range group {
			current = append(current, line)
			y -= line.style.leadingHeight()
		}
		i += len(group)
	}
	if len(current) > 0 || len(pages) == 0 {
		pages = append(pages, current)
	}
	return pages
}

// wrapLine splits prose at spaces using Helvetica metrics. The trailing space
// stays with the preceding fragment, so the original line is recoverable by
// concatenating consecutive literal strings.
func wrapLine(text string, maxWidth float64, size float64) []string {
	if text == "" || textWidth(text, size) <= maxWidth {
		return []string{text}
	}
	var fragments []string
	var current strings.Builder
	for _, word := range strings.SplitAfter(text, " ") {
		if current.Len() > 0 && textWidth(current.String()+word, size) > maxWidth {
			fragments = append(fragments, current.String())
			current.Reset()
		}
		current.WriteString(word)
	}
	if current.Len() > 0 {
		fragments = append(fragments, current.String())
	}
	return fragments
}

var helveticaUpper = [...]int{
	667, 667, 722, 722, 667, 611, 778, 722, 278, 500, 667, 556, 833, 722, 778, 667,
	778, 722, 667, 611, 722, 667, 944, 667, 667, 611,
}

var helveticaLower = [...]int{
	556, 556, 500, 556, 556, 278, 556, 556, 222, 222, 500, 222, 833, 556, 556, 556,
	556, 333, 500, 278, 556, 500, 722, 500, 500, 500,
}

// glyphWidth returns the Helvetica AFM width in thousandths of an em.
func glyphWidth(r rune) int {
	switch {
	case r >= 'A' && r <= 'Z':
		return helveticaUpper[r-'A']
	case r >= 'a' && r <= 'z':
		return helveticaLower[r-'a']
	case r >= '0' && r <= '9':
		return 556
	}
	switch r {
	case '!', ' ', ',', '.', ':', ';', '(', ')', '[', ']', '/', '-':
		return 278
	case '"', '*':
		return 355
	case '#', '$', '=', '>', '<', '_':
		return 556
	case '%':
		return 889
	case '&':
		return 667
	case '\'':
		return 191
	case '+':
		return 584
	case '?':
		return 556
	case '@':
		return 1015
	case '^':
		return 469
	case '`':
		return 333
	case '{', '}':
		return 334
	case '|':
		return 260
	case '~':
		return 584
	default:
		// Names such as "Bogotá" fall back to the average glyph width.
		return 556
	}
}

func textWidth(text string, size float64) float64 {
	units := 0
	for _, r := range text {
		units += glyphWidth(r)
	}
	return float64(units) * size / 1000
}

// writePDF assembles the objects, xref table and trailer. Object 1 is the
// catalog, 2 the pages, 3-4 the two fonts, page objects start at 5 and their
// content streams follow all page objects.
func writePDF(pages [][]pdfLine) []byte {
	pageCount := len(pages)
	firstPage := 5
	firstStream := firstPage + pageCount

	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n")
	offsets := make([]int, firstStream+pageCount)
	offsets[0] = 0

	writeObject(&buf, offsets, 1, "<< /Type /Catalog /Pages 2 0 R >>")
	kids := make([]string, pageCount)
	for i := 0; i < pageCount; i++ {
		kids[i] = strconv.Itoa(firstPage+i) + " 0 R"
	}
	writeObject(&buf, offsets, 2, "<< /Type /Pages /Kids ["+strings.Join(kids, " ")+"] /Count "+
		strconv.Itoa(pageCount)+" >>")
	writeObject(&buf, offsets, 3, "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica /Encoding /WinAnsiEncoding >>")
	writeObject(&buf, offsets, 4, "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica-Bold /Encoding /WinAnsiEncoding >>")

	for i, lines := range pages {
		pageNumber := firstPage + i
		streamNumber := firstStream + i
		writeObject(&buf, offsets, pageNumber,
			"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 "+strconv.Itoa(pageWidth)+" "+
				strconv.Itoa(pageHeight)+"] /Resources << /Font << /F1 3 0 R /F2 4 0 R >> >> /Contents "+
				strconv.Itoa(streamNumber)+" 0 R >>")
		writeStream(&buf, offsets, streamNumber, lines, i+1, pageCount)
	}

	xrefOffset := buf.Len()
	size := firstStream + pageCount
	buf.WriteString("xref\n0 ")
	buf.WriteString(strconv.Itoa(size))
	buf.WriteString("\n0000000000 65535 f\r\n")
	for i := 1; i < size; i++ {
		fmt.Fprintf(&buf, "%010d 00000 n\r\n", offsets[i])
	}
	fmt.Fprintf(&buf, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", size, xrefOffset)
	return buf.Bytes()
}

func writeObject(buf *bytes.Buffer, offsets []int, number int, body string) {
	offsets[number] = buf.Len()
	fmt.Fprintf(buf, "%d 0 obj\n%s\nendobj\n", number, body)
}

func writeStream(buf *bytes.Buffer, offsets []int, number int, lines []pdfLine, pageNumber, pageCount int) {
	var content bytes.Buffer
	drawPageChrome(&content)
	y := float64(contentTop)
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		font, size, leading, color := line.style.metrics()
		if line.style == styleSection {
			fmt.Fprintf(&content, "q\n%s RG\n0.6 w\n%d %.2f m %d %.2f l S\nQ\n",
				colorBlue, textX, y+7, pageWidth-cardX, y+7)
		}
		if line.block != 0 {
			height := 0.0
			for j := i; j < len(lines) && lines[j].block == line.block; j++ {
				height += lines[j].style.leadingHeight()
			}
			fmt.Fprintf(&content, "q\n%s rg\n48 %.2f 520 %.2f re f\n%s rg\n48 %.2f 2 %.2f re f\nQ\n",
				colorApron, y-height+2, height, colorBlue, y-height+2, height)
		}
		x := textX
		if line.style == styleCanonical {
			x = canonicalX
		}
		fmt.Fprintf(&content, "BT\n%s %s Tf\n%s rg\n1 0 0 1 %d %.2f Tm\n%s Tj\nET\n",
			font, size, color, x, y, pdfString(line.text))
		y -= leading
	}
	fmt.Fprintf(&content, "q\n%s rg\n%d %d %d %d re f\nQ\n", colorAsphalt, cardX, cardY, cardWidth, 24)
	fmt.Fprintf(&content, "BT\n/F1 8 Tf\n%s rg\n1 0 0 1 %d %d Tm\n%s Tj\nET\n",
		colorOnAsphalt, textX, 45, pdfString(fmt.Sprintf("Page %d of %d", pageNumber, pageCount)))

	offsets[number] = buf.Len()
	fmt.Fprintf(buf, "%d 0 obj\n<< /Length %d >>\nstream\n", number, content.Len())
	buf.Write(content.Bytes())
	buf.WriteString("endstream\nendobj\n")
}

// drawPageChrome paints every page with the same apron background, surface
// card, asphalt title band, yellow accent rule and bold white title.
func drawPageChrome(content *bytes.Buffer) {
	fmt.Fprintf(content, "q\n%s rg\n0 0 %d %d re f\n", colorApron, pageWidth, pageHeight)
	fmt.Fprintf(content, "%s rg\n%s RG\n0.6 w\n%d %d %d %d re f\n%d %d %d %d re S\n",
		colorSurface, colorLine, cardX, cardY, cardWidth, cardHeight, cardX, cardY, cardWidth, cardHeight)
	fmt.Fprintf(content, "%s rg\n%d %d %d %d re f\n", colorAsphalt, cardX, headerY, cardWidth, headerHeight)
	fmt.Fprintf(content, "%s rg\n%d %d %d 3 re f\nQ\n", colorYellow, cardX, headerY-2, cardWidth)
	fmt.Fprintf(content, "BT\n/F2 16 Tf\n%s rg\n1 0 0 1 %d %d Tm\n%s Tj\nET\n",
		colorOnAsphalt, textX, 721, pdfString("HUBX traceability evidence"))
}

// pdfString escapes a line as a PDF literal string. Non-ASCII runes are
// written as Latin-1 octal escapes (with '?' for runes outside Latin-1), so
// the document uses the Helvetica WinAnsi encoding without embedding a font.
func pdfString(line string) string {
	var out strings.Builder
	out.WriteByte('(')
	for _, r := range line {
		switch {
		case r == '\\' || r == '(' || r == ')':
			out.WriteByte('\\')
			out.WriteRune(r)
		case r == '\n':
			out.WriteString("\\n")
		case r == '\r':
			out.WriteString("\\r")
		case r == '\t':
			out.WriteString("\\t")
		case r < 32 || r > 126:
			if r <= 255 {
				fmt.Fprintf(&out, "\\%03o", r)
			} else {
				out.WriteRune('?')
			}
		default:
			out.WriteRune(r)
		}
	}
	out.WriteByte(')')
	return out.String()
}
