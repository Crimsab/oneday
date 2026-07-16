import arFlag from "flag-icons/flags/4x3/sa.svg";
import csFlag from "flag-icons/flags/4x3/cz.svg";
import daFlag from "flag-icons/flags/4x3/dk.svg";
import deFlag from "flag-icons/flags/4x3/de.svg";
import elFlag from "flag-icons/flags/4x3/gr.svg";
import enFlag from "flag-icons/flags/4x3/gb.svg";
import esFlag from "flag-icons/flags/4x3/es.svg";
import fiFlag from "flag-icons/flags/4x3/fi.svg";
import frFlag from "flag-icons/flags/4x3/fr.svg";
import hiFlag from "flag-icons/flags/4x3/in.svg";
import idFlag from "flag-icons/flags/4x3/id.svg";
import itFlag from "flag-icons/flags/4x3/it.svg";
import jaFlag from "flag-icons/flags/4x3/jp.svg";
import koFlag from "flag-icons/flags/4x3/kr.svg";
import nlFlag from "flag-icons/flags/4x3/nl.svg";
import noFlag from "flag-icons/flags/4x3/no.svg";
import plFlag from "flag-icons/flags/4x3/pl.svg";
import ptFlag from "flag-icons/flags/4x3/pt.svg";
import roFlag from "flag-icons/flags/4x3/ro.svg";
import ruFlag from "flag-icons/flags/4x3/ru.svg";
import svFlag from "flag-icons/flags/4x3/se.svg";
import thFlag from "flag-icons/flags/4x3/th.svg";
import trFlag from "flag-icons/flags/4x3/tr.svg";
import ukFlag from "flag-icons/flags/4x3/ua.svg";
import viFlag from "flag-icons/flags/4x3/vn.svg";
import zhFlag from "flag-icons/flags/4x3/cn.svg";

const COMMON_LANGUAGE_CODES = ["en", "it", "fr", "de", "es", "pt", "nl", "pl", "ro", "sv", "da", "no", "fi", "cs", "el", "tr", "uk", "ru", "ar", "hi", "id", "vi", "th", "ja", "ko", "zh"] as const;

export interface LanguageOption { code: string; name: string }

const FLAG_BY_LANGUAGE: Record<(typeof COMMON_LANGUAGE_CODES)[number], string> = {
  en: enFlag, it: itFlag, fr: frFlag, de: deFlag, es: esFlag, pt: ptFlag, nl: nlFlag, pl: plFlag,
  ro: roFlag, sv: svFlag, da: daFlag, no: noFlag, fi: fiFlag, cs: csFlag, el: elFlag, tr: trFlag,
  uk: ukFlag, ru: ruFlag, ar: arFlag, hi: hiFlag, id: idFlag, vi: viFlag, th: thFlag, ja: jaFlag,
  ko: koFlag, zh: zhFlag,
};

export function languageFlagUrl(code: string): string | undefined {
  return FLAG_BY_LANGUAGE[code.toLowerCase() as keyof typeof FLAG_BY_LANGUAGE];
}

export function languageCatalog(locale: string): LanguageOption[] {
  const names = new Intl.DisplayNames([locale], { type: "language" });
  return COMMON_LANGUAGE_CODES.map((code) => ({ code, name: names.of(code) || code.toUpperCase() }))
    .sort((left, right) => left.name.localeCompare(right.name, locale));
}
