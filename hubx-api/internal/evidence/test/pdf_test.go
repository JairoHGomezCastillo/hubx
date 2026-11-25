package test

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"hubx-api/internal/evidence"
)

var pdfLiteral = regexp.MustCompile(`\(((?:\\.|[^\\()])*)\)`)

// pdfLines decodes the uncompressed literal strings of the PDF content stream
// exactly as an external recipient would read them.
func pdfLines(t *testing.T, data []byte) []string {
	t.Helper()
	matches := pdfLiteral.FindAllSubmatch(data, -1)
	lines := make([]string, 0, len(matches))
	for _, match := range matches {
		lines = append(lines, decodeLiteral(t, string(match[1])))
	}
	return lines
}

func decodeLiteral(t *testing.T, literal string) string {
	t.Helper()
	var out strings.Builder
	for i := 0; i < len(literal); i++ {
		if literal[i] != '\\' {
			out.WriteByte(literal[i])
			continue
		}
		i++
		if i >= len(literal) {
			t.Fatalf("trailing backslash in PDF literal %q", literal)
		}
		switch literal[i] {
		case 'n':
			out.WriteByte('\n')
		case 'r':
			out.WriteByte('\r')
		case 't':
			out.WriteByte('\t')
		case '(', ')', '\\':
			out.WriteByte(literal[i])
		default:
			if literal[i] < '0' || literal[i] > '7' {
				t.Fatalf("unsupported escape \\%c in PDF literal %q", literal[i], literal)
			}
			octal := string(literal[i])
			for len(octal) < 3 && i+1 < len(literal) && literal[i+1] >= '0' && literal[i+1] <= '7' {
				i++
				octal += string(literal[i])
			}
			value, err := strconv.ParseInt(octal, 8, 16)
			if err != nil {
				t.Fatalf("invalid octal escape %q: %v", octal, err)
			}
			out.WriteByte(byte(value))
		}
	}
	raw := []byte(out.String())
	text := string(raw)
	if strings.ContainsRune(text, '\uFFFD') {
		// Same fallback as the QA PDF reader: Latin-1 names remain readable.
		decoded := make([]rune, len(raw))
		for i, value := range raw {
			decoded[i] = rune(value)
		}
		return string(decoded)
	}
	return text
}

func fullEvidence() *evidence.Evidence {
	generated := time.Date(2026, 11, 25, 9, 30, 0, 0, time.UTC)
	return &evidence.Evidence{
		Shipment: evidence.ShipmentData{
			Code:         "HBX-TEST01",
			Origin:       "Bogotá",
			Destination:  "Miami",
			Flight:       "AV-204",
			WeightKg:     "100.0",
			CustomerName: "Distribuciones Andina",
			Stage:        "Assigned to partner",
		},
		Updates: []evidence.UpdateEvidence{
			{
				Sequence:            1,
				Stage:               "In flight",
				ResponsibleParty:    "Laura Operator",
				RecordedAt:          time.Date(2026, 11, 21, 10, 15, 30, 123456000, time.UTC),
				Fingerprint:         "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				PreviousFingerprint: "-",
				CanonicalInput: strings.Join([]string{
					"shipment_id=11111111-1111-4111-8111-111111111111",
					"sequence=1",
					"stage=In flight",
					"responsible_party_id=33333333-3333-4333-8333-333333333333",
					"assigned_partner_id=-",
					"recorded_at=2026-11-21T10:15:30.123456Z",
					"previous_fingerprint=-",
				}, "\n"),
			},
			{
				Sequence:            2,
				Stage:               "At destination airport",
				ResponsibleParty:    "Laura Operator",
				AssignedPartner:     "Transportes Rápido",
				RecordedAt:          time.Date(2026, 11, 22, 11, 20, 0, 0, time.UTC),
				Fingerprint:         "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
				PreviousFingerprint: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				CanonicalInput: strings.Join([]string{
					"shipment_id=11111111-1111-4111-8111-111111111111",
					"sequence=2",
					"stage=At destination airport",
					"responsible_party_id=33333333-3333-4333-8333-333333333333",
					"assigned_partner_id=44444444-4444-4444-8444-444444444444",
					"recorded_at=2026-11-22T11:20:00Z",
					"previous_fingerprint=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				}, "\n"),
			},
		},
		GeneratedAt:  generated,
		Instructions: evidence.Instructions,
	}
}

func TestHU10_RenderIncludesShipmentDataChainAndInstructions(t *testing.T) {
	data := evidence.Render(fullEvidence())
	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		t.Fatalf("Render() does not start with the PDF magic bytes: %q", data[:min(16, len(data))])
	}

	text := strings.Join(pdfLines(t, data), "\n")
	for _, want := range []string{
		"HUBX traceability evidence",
		"Code: HBX-TEST01",
		"Origin: Bogotá",
		"Destination: Miami",
		"Flight: AV-204",
		"Weight: 100.0 kg",
		"Customer: Distribuciones Andina",
		"Current stage: Assigned to partner",
		"Updates:",
		"1. In flight",
		"2. At destination airport",
		"Laura Operator",
		"Transportes Rápido",
		"Generated at: 2026-11-25T09:30:00Z",
		"How to verify the fingerprints:",
		"Scope:",
		"consistent with itself",
		"did not rebuild the whole history",
		"Fingerprint: aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"Fingerprint: bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("PDF text does not contain %q", want)
		}
	}

	canonical := regexp.MustCompile(`shipment_id=[^\n]+\nsequence=[1-9][0-9]*\nstage=[^\n]+\nresponsible_party_id=[^\n]+\nassigned_partner_id=[^\n]+\nrecorded_at=[^\n]+\nprevious_fingerprint=[^\n]+`)
	blocks := canonical.FindAllString(text, -1)
	if len(blocks) != 2 {
		t.Fatalf("canonical blocks = %d, want 2; text = %q", len(blocks), text)
	}
	for i, update := range fullEvidence().Updates {
		if blocks[i] != update.CanonicalInput {
			t.Errorf("canonical block %d = %q, want %q", i, blocks[i], update.CanonicalInput)
		}
	}
	flightIndex := strings.Index(text, "1. In flight")
	airportIndex := strings.Index(text, "2. At destination airport")
	if flightIndex < 0 || airportIndex < 0 || flightIndex > airportIndex {
		t.Errorf("updates are not in chain order: In flight at %d, At destination airport at %d", flightIndex, airportIndex)
	}
}

func TestHU10_RenderStatesEmptyChain(t *testing.T) {
	doc := fullEvidence()
	doc.Updates = nil
	text := strings.Join(pdfLines(t, evidence.Render(doc)), "\n")
	if !strings.Contains(text, "The shipment has no updates yet.") {
		t.Errorf("PDF text does not state the empty chain: %q", text)
	}
	if strings.Contains(text, "shipment_id=") {
		t.Error("empty chain must not print canonical update blocks")
	}
}

func TestHU10_RenderEscapesPDFStrings(t *testing.T) {
	doc := fullEvidence()
	doc.Shipment.Code = `HBX-(TEST)\X`
	data := evidence.Render(doc)
	for _, escaped := range []string{`\(`, `\)`, `\\`} {
		if !bytes.Contains(data, []byte(escaped)) {
			t.Errorf("PDF stream does not escape %q", escaped)
		}
	}
	text := strings.Join(pdfLines(t, data), "\n")
	if !strings.Contains(text, "Code: "+doc.Shipment.Code) {
		t.Errorf("decoded PDF text does not keep the original code: %q", text)
	}
}

func TestHU10_RenderHasValidXref(t *testing.T) {
	data := evidence.Render(fullEvidence())
	xref := regexp.MustCompile(`(?m)^xref\r?\n0 (\d+)\r?\n`)
	match := xref.FindSubmatch(data)
	if match == nil {
		t.Fatal("PDF has no xref table")
	}
	count, err := strconv.Atoi(string(match[1]))
	if err != nil || count < 2 {
		t.Fatalf("xref size = %q, want at least 2", match[1])
	}
	entries := regexp.MustCompile(`(?m)^(\d{10}) 00000 n\r?$`).FindAllSubmatch(data, -1)
	if len(entries) != count-1 {
		t.Fatalf("xref entries = %d, want %d", len(entries), count-1)
	}
	for i, entry := range entries {
		offset, err := strconv.Atoi(string(entry[1]))
		if err != nil {
			t.Fatalf("invalid xref offset %q: %v", entry[1], err)
		}
		if offset >= len(data) {
			t.Fatalf("xref offset %d is beyond the document (%d bytes)", offset, len(data))
		}
		want := strconv.Itoa(i+1) + " 0 obj"
		if !bytes.HasPrefix(data[offset:], []byte(want)) {
			t.Errorf("xref entry %d points to %q, want %q", i+1, data[offset:min(offset+12, len(data))], want)
		}
	}
	if !bytes.Contains(data, []byte("trailer")) || !bytes.HasSuffix(bytes.TrimSpace(data), []byte("%%EOF")) {
		t.Error("PDF has no valid trailer or final EOF marker")
	}
}

// The MFE palette comes from hubx-mfe/src/index.css. The PDF must use the same
// colors, expressed as normalized RGB PDF operators, and both standard fonts.
func TestHU10_RenderUsesHubxVisualStyle(t *testing.T) {
	data := evidence.Render(fullEvidence())
	for _, want := range []string{
		"0.910 0.929 0.933 rg", // --apron: page background
		"1 1 1 rg",             // --surface: card and title
		"0.776 0.816 0.824 RG", // --line: card border
		"0.188 0.224 0.235 rg", // --asphalt: header and footer bands
		"0.969 0.741 0.086 rg", // --yellow: accent rule
		"0.110 0.310 0.769 rg", // --blue: labels and section rules
		"/F1 3 0 R /F2 4 0 R",
		"/BaseFont /Helvetica-Bold",
		" re f",
		" re S",
		"Page 1 of ",
	} {
		if !bytes.Contains(data, []byte(want)) {
			t.Errorf("PDF stream does not contain visual operator %q", want)
		}
	}
	if got := len(pdfLiteral.FindAll(data, -1)); got < 20 {
		t.Errorf("PDF has only %d literal strings, expected a laid-out document", got)
	}
}

// runLength reports how many consecutive literal strings concatenate to want;
// wrapped prose keeps its spaces at the end of each fragment, so the original
// instruction line is recoverable without any separator.
func runLength(lines []string, want string) int {
	for start := range lines {
		joined := ""
		for end := start; end < len(lines) && len(joined) < len(want); end++ {
			joined += lines[end]
			if joined == want {
				return end - start + 1
			}
		}
	}
	return 0
}

func TestHU10_RenderWrapsProseButNotCanonicalLines(t *testing.T) {
	doc := fullEvidence()
	lines := pdfLines(t, evidence.Render(doc))
	wrapped := 0
	for _, instruction := range strings.Split(doc.Instructions, "\n") {
		parts := runLength(lines, instruction)
		if parts == 0 {
			t.Errorf("instruction line is lost or altered: %q", instruction)
			continue
		}
		if parts > 1 {
			wrapped++
		}
	}
	if wrapped == 0 {
		t.Error("no long instruction line was wrapped across PDF literal strings")
	}
	for _, line := range lines {
		// Canonical key=value lines and fingerprints are intentionally long;
		// human-readable prose must fit the page width.
		if strings.Contains(line, "=") || strings.Contains(line, "Fingerprint:") || strings.Contains(line, "Previous:") {
			continue
		}
		if len(line) > 95 {
			t.Errorf("prose line is too long for the page (%d chars): %q", len(line), line)
		}
	}
	text := strings.Join(lines, "\n")
	for _, want := range []string{"consistent with itself", "did not rebuild the whole history"} {
		if !strings.Contains(text, want) {
			t.Errorf("PDF text does not keep scope phrase %q intact", want)
		}
	}
}

// The names statement (contract success item 6) follows the two Scope sentences;
// each sentence must be one whole literal string, never wrapped, also with an
// empty chain.
func TestHU10_RenderStatesNamesAreNotCoveredByFingerprints(t *testing.T) {
	statement := []string{
		"Fingerprints cover the party identifiers, not the printed names.",
		"The covered fields are responsible_party_id and assigned_partner_id.",
		"The names cannot be verified from this document alone.",
	}
	empty := fullEvidence()
	empty.Updates = nil
	for name, doc := range map[string]*evidence.Evidence{"with updates": fullEvidence(), "empty chain": empty} {
		t.Run(name, func(t *testing.T) {
			lines := pdfLines(t, evidence.Render(doc))
			last := -1
			for i, line := range lines {
				if line == "It does not by itself prove that HUBX did not rebuild the whole history." {
					last = i
				}
			}
			if last < 0 {
				t.Fatal("second Scope sentence is missing as one whole literal string")
			}
			for offset, sentence := range statement {
				index := last + 1 + offset
				if index >= len(lines) || lines[index] != sentence {
					got := "<end of document>"
					if index < len(lines) {
						got = lines[index]
					}
					t.Errorf("literal after Scope #%d = %q, want whole sentence %q", offset+1, got, sentence)
				}
			}
		})
	}
}

func assertCanonicalBlocks(t *testing.T, lines []string, updates []evidence.UpdateEvidence) {
	t.Helper()
	for _, update := range updates {
		want := strings.Split(update.CanonicalInput, "\n")
		// shipment_id repeats between updates; locate the block by its unique
		// sequence line and step back to the first of the seven key=value lines.
		start := -1
		for i := range lines {
			if i > 0 && lines[i] == want[1] {
				start = i - 1
				break
			}
		}
		if start < 0 {
			t.Fatalf("canonical block of update %d is missing", update.Sequence)
		}
		for offset, expected := range want {
			if got := lines[start+offset]; got != expected {
				t.Errorf("canonical block %d line %d = %q, want %q", update.Sequence, offset, got, expected)
			}
		}
		if got := lines[start+len(want)]; got != "Fingerprint: "+update.Fingerprint {
			t.Errorf("line after canonical block %d = %q, want its fingerprint", update.Sequence, got)
		}
		if got := lines[start+len(want)+1]; got != "Previous: "+update.PreviousFingerprint {
			t.Errorf("second line after canonical block %d = %q, want its previous fingerprint", update.Sequence, got)
		}
	}
}

func TestHU10_RenderKeepsCanonicalBlocksAsConsecutiveStrings(t *testing.T) {
	doc := fullEvidence()
	assertCanonicalBlocks(t, pdfLines(t, evidence.Render(doc)), doc.Updates)
}

func TestHU10_RenderRepeatsChromeAndNeverSplitsBlocksAcrossPages(t *testing.T) {
	doc := fullEvidence()
	// Nine updates force several pages with the QA-shaped document.
	for sequence := 3; sequence <= 10; sequence++ {
		update := doc.Updates[1]
		update.Sequence = sequence
		update.CanonicalInput = strings.Replace(update.CanonicalInput, "sequence=2",
			"sequence="+strconv.Itoa(sequence), 1)
		update.Fingerprint = strings.Repeat(strconv.Itoa(sequence), 64)
		doc.Updates = append(doc.Updates, update)
	}
	data := evidence.Render(doc)
	pages := len(regexp.MustCompile(`/Type /Page\b`).FindAll(data, -1))
	if pages < 2 {
		t.Fatalf("document with nine updates has %d page(s), want several", pages)
	}
	text := strings.Join(pdfLines(t, data), "\n")
	footers := len(regexp.MustCompile(`Page \d+ of \d+`).FindAllString(text, -1))
	titles := strings.Count(text, "HUBX traceability evidence")
	if footers != pages || titles != pages {
		t.Errorf("chrome is not repeated: %d pages, %d footers, %d titles", pages, footers, titles)
	}
	for page := 1; page <= pages; page++ {
		want := fmt.Sprintf("Page %d of %d", page, pages)
		if !strings.Contains(text, want) {
			t.Errorf("footer %q is missing", want)
		}
	}
	assertCanonicalBlocks(t, pdfLines(t, data), doc.Updates)
}
