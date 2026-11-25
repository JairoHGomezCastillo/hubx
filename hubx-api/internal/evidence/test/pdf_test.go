package test

import (
	"bytes"
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
