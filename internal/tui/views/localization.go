package views

import appi18n "github.com/crimsab/oneday/internal/i18n"

func viewLocalizer(localizers []appi18n.Localizer) appi18n.Localizer {
	if len(localizers) > 0 {
		return localizers[0]
	}
	return appi18n.New(appi18n.English)
}
