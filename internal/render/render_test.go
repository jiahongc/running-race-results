// internal/render/render_test.go
package render

import (
	"bytes"
	"strings"
	"testing"

	"github.com/jiahongchen/race-results/internal/domain"
)

func sample() domain.Result {
	return domain.Result{Provider: "mika", RaceName: "BMW Berlin Marathon", Year: 2025,
		Runner: "Jane Doe", Bib: "1234", NetTime: "02:45:10", OverallPlace: 512}
}

func TestTableContainsCoreFields(t *testing.T) {
	var b bytes.Buffer
	if err := Table(&b, sample()); err != nil {
		t.Fatal(err)
	}
	out := b.String()
	for _, want := range []string{"Jane Doe", "1234", "02:45:10", "Berlin"} {
		if !strings.Contains(out, want) {
			t.Fatalf("table missing %q in:\n%s", want, out)
		}
	}
}

func TestJSONRoundTrips(t *testing.T) {
	var b bytes.Buffer
	if err := JSON(&b, sample()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(b.String(), `"Runner": "Jane Doe"`) {
		t.Fatalf("json missing runner:\n%s", b.String())
	}
}

func TestTablePlaces(t *testing.T) {
	r := domain.Result{Provider: "athlinks", RaceName: "R", Year: 2024, Runner: "A B", Bib: "7",
		OverallPlace: 3, GenderPlace: 2, AgeGroup: "M30-39", AgeGroupPlace: 1}
	var b bytes.Buffer
	if err := Table(&b, r); err != nil {
		t.Fatal(err)
	}
	out := b.String()
	for _, want := range []string{"Gender place", "Age group", "M30-39", "Age group place"} {
		if !strings.Contains(out, want) {
			t.Fatalf("table missing %q in:\n%s", want, out)
		}
	}
}

func TestTableOmitsZeroPlaces(t *testing.T) {
	r := domain.Result{Provider: "x", RaceName: "R", Year: 2024, Runner: "A B", Bib: "9"}
	var b bytes.Buffer
	if err := Table(&b, r); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(b.String(), "Gender place") {
		t.Fatalf("table showed a zero gender place:\n%s", b.String())
	}
}

func TestTableOmitsZeroYear(t *testing.T) {
	r := domain.Result{Provider: "x", RaceName: "Some Race", Runner: "A B", Bib: "9"}
	var b bytes.Buffer
	if err := Table(&b, r); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(b.String(), "Some Race 0") {
		t.Fatalf("table faked a zero year:\n%s", b.String())
	}
	if !strings.Contains(b.String(), "Some Race") {
		t.Fatalf("table missing race name:\n%s", b.String())
	}
}
