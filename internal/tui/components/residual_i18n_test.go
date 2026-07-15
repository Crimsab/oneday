package components

import (
	"strings"
	"testing"

	"github.com/crimsab/oneday/internal/i18n"
)

func TestItalianOverlayAndAchievementChrome(t *testing.T) {
	t.Parallel()
	loc := i18n.New(i18n.Italian)

	overlay := NewOverlay(loc)
	overlay.SetSize(80, 24)
	overlay.Show("Titolo", "Contenuto")
	if view := overlay.View(); !strings.Contains(view, "chiudere") || strings.Contains(view, "to close") {
		t.Fatalf("overlay chrome was not localized: %q", view)
	}

	popup := NewAchievementPopup(loc)
	popup.SetSize(80, 24)
	popup.Show("Esploratore", "Hai scoperto un luogo.", "rare", "exploration")
	if view := popup.View(); !strings.Contains(view, "TRAGUARDO SBLOCCATO") || strings.Contains(view, "ACHIEVEMENT UNLOCKED") {
		t.Fatalf("achievement chrome was not localized: %q", view)
	}
}
