package cleanup

import "testing"

func TestParseSize(t *testing.T) {
	cases := map[string]float64{
		"503.9MB":                          503.9e6,
		"3.4 GB":                           3.4e9,
		"800kB":                            800e3,
		"1.5GB":                            1.5e9,
		"200MB":                            200e6,
		"1GiB":                             1 << 30,
		"would free approximately 503.9MB": 503.9e6,
	}
	for in, want := range cases {
		got, ok := parseSize(in)
		if !ok {
			t.Errorf("parseSize(%q) failed", in)
			continue
		}
		if got != want {
			t.Errorf("parseSize(%q) = %v, want %v", in, got, want)
		}
	}
	if _, ok := parseSize("no size here"); ok {
		t.Error("parseSize of non-size should fail")
	}
}

func TestFormatBytes(t *testing.T) {
	cases := map[float64]string{
		500:           "500B",
		1500:          "1.5KB",
		503.9e6:       "503.9MB",
		1.2e9:         "1.2GB",
		5_120_000_000: "5.1GB",
	}
	for in, want := range cases {
		if got := formatBytes(in); got != want {
			t.Errorf("formatBytes(%v) = %q, want %q", in, got, want)
		}
	}
}

func TestFormatKB(t *testing.T) {
	// 800000 KB == 819.2 MB.
	if got := formatKB(800000); got != "819.2MB" {
		t.Errorf("formatKB(800000) = %q, want 819.2MB", got)
	}
}

func TestDockerReclaimableHandlesMultiWordTypes(t *testing.T) {
	out := `TYPE            TOTAL     ACTIVE    SIZE      RECLAIMABLE
Images          5         2         2.0GB     1.5GB (75%)
Containers      3         1         100MB     50MB (50%)
Local Volumes   2         1         500MB     200MB (40%)
Build Cache     10        0         300MB     300MB`
	total, ok := dockerReclaimable(out)
	if !ok {
		t.Fatal("expected reclaimable figures")
	}
	want := 1.5e9 + 50e6 + 200e6 + 300e6
	if total != want {
		t.Errorf("dockerReclaimable = %v, want %v", total, want)
	}
}

func TestDockerReclaimableNoHeader(t *testing.T) {
	if _, ok := dockerReclaimable("garbage output"); ok {
		t.Error("missing RECLAIMABLE header should yield ok=false")
	}
}
