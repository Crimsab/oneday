import { afterEach, describe, expect, it } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";
import { setInterfaceLocale } from "../i18n";
import type { StorySnapshot, VisualAsset } from "../types";
import { CanonicalMap } from "./CanonicalMap";
import { CommandPalette } from "./CommandPalette";
import { CustomSelect } from "./CustomSelect";
import { StoryPath } from "./StoryPath";

describe("localized navigation surfaces", () => {
  afterEach(async () => {
    await setInterfaceLocale("en");
  });

  it("renders Italian map presentation while preserving canonical place values", async () => {
    await setInterfaceLocale("it-IT");
    const html = renderToStaticMarkup(
      <CanonicalMap
        locationsValue={[{ id: "porto-ombra", name: "Porto Ombra", kind: "place" }]}
        edgesValue={[]}
        currentLocationId="porto-ombra"
      />,
    );

    expect(html).toContain("Gerarchia della mappa");
    expect(html).toContain("Mappa interattiva di Mondo");
    expect(html).toContain("Porto Ombra, luogo, luogo attuale");
    expect(html).toContain("Porto Ombra");
    expect(html).not.toContain("Map controls");
  });

  it("localizes story path controls without changing story content", async () => {
    await setInterfaceLocale("it");
    const snapshot = {
      world: {
        current_location: "Porto Ombra, Molo Nord",
        current_chapter: 2,
        world_time: { minute_of_day: 480 },
      },
    } as unknown as StorySnapshot;
    const locationAsset = {
      status: "queued",
      prompt: "",
      subject: "Porto Ombra",
    } as unknown as VisualAsset;
    const html = renderToStaticMarkup(
      <StoryPath
        snapshot={snapshot}
        locationAsset={locationAsset}
        paused
        onTogglePaused={() => undefined}
        onClearTranscript={() => undefined}
      />,
    );

    expect(html).toContain("Percorso attuale della storia");
    expect(html).toContain("Capitolo 2");
    expect(html).toContain("Mattina");
    expect(html).toContain("Porto Ombra, Molo Nord");
    expect(html).toContain("Riprendi gli aggiornamenti");
    expect(html).toContain("Immagine della scena: in coda");
  });

  it("localizes palette chrome and generic select accessibility while preserving command tokens and option values", async () => {
    await setInterfaceLocale("it");
    const palette = renderToStaticMarkup(
      <CommandPalette
        items={[{
          name: "/advance",
          hint: "Passa al prossimo momento significativo.",
          aliases: ["/avanza"],
          value: "/advance ",
          group: "play",
          kind: "command",
          badge: "Terminal",
        }]}
        activeIndex={0}
        variant="full"
        onActiveIndexChange={() => undefined}
        onPick={() => undefined}
      />,
    );
    const select = renderToStaticMarkup(
      <CustomSelect
        value="canonical-id"
        options={[{ value: "canonical-id", label: "Voce canonica", iconSrc: "/flags/example.svg" }]}
        onChange={() => undefined}
      />,
    );

    expect(palette).toContain("Elenco comandi");
    expect(palette).toContain("Invio");
    expect(palette).toContain("Terminale");
    expect(palette).toContain("/advance");
    expect(select).toContain('aria-label="Seleziona un’opzione"');
    expect(select).toContain("Voce canonica");
    expect(select).toContain('src="/flags/example.svg"');
    expect(select).toContain('alt=""');
  });
});
