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
