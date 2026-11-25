package evidence

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// linesPerPage leaves room for the 10 pt Helvetica text of the evidence
// document inside a US Letter page (12 pt leading, 40 pt margins).
const linesPerPage = 60

// Render writes a self-contained, uncompressed PDF document. Every document
// line is emitted as one literal PDF string so a recipient can read the text
// directly from the file; no font embedding or external resource is needed.
func Render(doc *Evidence) []byte {
	lines := documentLines(doc)
	pages := make([][]string, 0, (len(lines)+linesPerPage-1)/linesPerPage)
	for start := 0; start < len(lines); start += linesPerPage {
		end := start + linesPerPage
		if end > len(lines) {
			end = len(lines)
		}
		pages = append(pages, lines[start:end])
	}
	if len(pages) == 0 {
		pages = [][]string{{}}
	}
	return writePDF(pages)
}

func documentLines(doc *Evidence) []string {
	lines := []string{
		"HUBX traceability evidence",
		"Code: " + doc.Shipment.Code,
		"Origin: " + doc.Shipment.Origin,
		"Destination: " + doc.Shipment.Destination,
		"Flight: " + doc.Shipment.Flight,
		"Weight: " + doc.Shipment.WeightKg + " kg",
		"Customer: " + doc.Shipment.CustomerName,
		"Current stage: " + doc.Shipment.Stage,
		"Updates:",
	}
	if len(doc.Updates) == 0 {
		lines = append(lines, "The shipment has no updates yet.")
	}
	for _, update := range doc.Updates {
		lines = append(lines,
			fmt.Sprintf("%d. %s", update.Sequence, update.Stage),
			"Responsible party: "+update.ResponsibleParty,
		)
		if update.AssignedPartner != "" {
			lines = append(lines, "Assigned partner: "+update.AssignedPartner)
		}
		lines = append(lines, "Recorded at: "+update.RecordedAt.UTC().Format(time.RFC3339Nano))
		// The canonical seven lines are the exact HU-03 fingerprint input.
		for _, line := range strings.Split(update.CanonicalInput, "\n") {
			lines = append(lines, line)
		}
		lines = append(lines,
			"Fingerprint: "+update.Fingerprint,
			// Deliberately not "Previous fingerprint": an external checker must
			// not mistake the predecessor link for the update's own fingerprint.
			"Previous: "+update.PreviousFingerprint,
		)
	}
	lines = append(lines, "Generated at: "+doc.GeneratedAt.UTC().Format(time.RFC3339))
	lines = append(lines, strings.Split(doc.Instructions, "\n")...)
	lines = append(lines,
		"Scope: This evidence proves that the shipment history is consistent with itself. "+
			"It does not by itself prove that HUBX did not rebuild the whole history.",
	)
	return lines
}

// writePDF assembles the objects, xref table and trailer. Page objects start at
// 4 (catalog, pages and font occupy 1-3); their streams follow the page objects.
func writePDF(pages [][]string) []byte {
	pageCount := len(pages)
	firstPage := 4
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

	for i, lines := range pages {
		pageNumber := firstPage + i
		streamNumber := firstStream + i
		writeObject(&buf, offsets, pageNumber,
			"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] "+
				"/Resources << /Font << /F1 3 0 R >> >> /Contents "+
				strconv.Itoa(streamNumber)+" 0 R >>")
		writeStream(&buf, offsets, streamNumber, lines)
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

func writeStream(buf *bytes.Buffer, offsets []int, number int, lines []string) {
	var content bytes.Buffer
	content.WriteString("BT\n/F1 10 Tf\n12 TL\n1 0 0 1 40 800 Tm\n")
	for _, line := range lines {
		content.WriteString(pdfString(line))
		content.WriteString(" Tj\nT*\n")
	}
	content.WriteString("ET\n")

	offsets[number] = buf.Len()
	fmt.Fprintf(buf, "%d 0 obj\n<< /Length %d >>\nstream\n", number, content.Len())
	buf.Write(content.Bytes())
	buf.WriteString("endstream\nendobj\n")
}

// pdfString escapes a line as a PDF literal string. Non-ASCII runes are
// written as Latin-1 octal escapes (with '?' for runes outside Latin-1), so
// the document uses the Helvetica WinAnsi encoding without embedding a font.
func pdfString(line string) string {
	var out strings.Builder
	out.WriteByte('(')
	for _, r := range line {
		c := byte('?')
		if r <= 255 {
			c = byte(r)
		}
		switch c {
		case '\\', '(', ')':
			out.WriteByte('\\')
			out.WriteByte(c)
		case '\n':
			out.WriteString("\\n")
		case '\r':
			out.WriteString("\\r")
		case '\t':
			out.WriteString("\\t")
		default:
			if c < 32 || c > 126 {
				fmt.Fprintf(&out, "\\%03o", c)
			} else {
				out.WriteByte(c)
			}
		}
	}
	out.WriteByte(')')
	return out.String()
}
