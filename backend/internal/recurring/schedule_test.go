package recurring

import (
	"testing"
	"time"
)

func TestDateForMonth(t *testing.T) {
	cases := []struct {
		name       string
		year       int
		month      time.Month
		dayOfMonth int
		wantDay    int
		wantMonth  time.Month
	}{
		{"jour normal en milieu de mois", 2026, time.March, 15, 15, time.March},
		{"31 dans un mois de 30 jours (avril)", 2026, time.April, 31, 30, time.April},
		{"31 en février non bissextile", 2026, time.February, 31, 28, time.February},
		{"31 en février bissextile", 2028, time.February, 31, 29, time.February},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := dateForMonth(c.year, c.month, c.dayOfMonth)

			if got.Day() != c.wantDay || got.Month() != c.wantMonth || got.Year() != c.year {
				t.Errorf("dateForMonth(%d, %s, %d) = %s, want day=%d month=%s",
					c.year, c.month, c.dayOfMonth, got.Format("2006-01-02"), c.wantDay, c.wantMonth)
			}
		})
	}
}

func TestAdvanceOneMonth(t *testing.T) {
	cases := []struct {
		name          string
		scheduledDate time.Time
		dayOfMonth    int
		wantYear      int
		wantMonth     time.Month
		wantDay       int
	}{
		{
			"31 janvier avance vers 28 février (non bissextile)",
			time.Date(2026, time.January, 31, 0, 0, 0, 0, time.UTC),
			31,
			2026, time.February, 28,
		},
		{
			"31 janvier avance vers 29 février (bissextile)",
			time.Date(2028, time.January, 31, 0, 0, 0, 0, time.UTC),
			31,
			2028, time.February, 29,
		},
		{
			"avance à cheval sur le nouvel an",
			time.Date(2026, time.December, 15, 0, 0, 0, 0, time.UTC),
			15,
			2027, time.January, 15,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := advanceOneMonth(c.scheduledDate, c.dayOfMonth)

			if got.Year() != c.wantYear || got.Month() != c.wantMonth || got.Day() != c.wantDay {
				t.Errorf("advanceOneMonth(%s, %d) = %s, want %04d-%02d-%02d",
					c.scheduledDate.Format("2006-01-02"), c.dayOfMonth, got.Format("2006-01-02"),
					c.wantYear, c.wantMonth, c.wantDay)
			}
		})
	}
}
