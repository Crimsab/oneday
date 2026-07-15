export const library = {
  filterStatus: "Stato delle storie", activeStories: "Attive ({{count}})", archivedStories: "Archiviate ({{count}})",
  storiesCount: "Storie ({{count}})", activeStoryCount_one: "1 storia attiva", activeStoryCount_other: "{{count}} storie attive",
  noArchivedStories: "Nessuna storia archiviata.", current: "Attuale", storyMeta: "{{genre}} · {{language}}", updated: "Aggiornata {{value}}",
  groups: { story: "Storia", character: "Personaggio", threads: "Trame attive", library: "Raccolta" },
  chapter: "Capitolo {{number}}", untitled: "Senza titolo", turn: "Turno {{turn}}", emptyNotes: "Seleziona una storia per vedere spunti, contatti e prossime piste.",
  name: "Nome", description: "Descrizione", genre: "Genere", tone: "Tono", language: "Lingua della storia", languageChangeConfirm: "Cambiare la lingua della storia? La modifica vale solo per i prossimi output. I turni esistenti restano invariati.", cancel: "Annulla", save: "Salva",
  storySummary: "Turno {{turn}} · {{genre}}", archived: "Archiviata", manage: "Gestisci {{name}}", edit: "Modifica", restore: "Ripristina", archive: "Archivia", delete: "Elimina", storyFallback: "Storia",
  note: { hook: "Spunto: {{value}}", front: "Fronte attivo: {{value}}", contact: "Contatto chiave: {{value}}", lead: "Prossima pista: {{value}}", updated: "Aggiornata: {{value}}" },
  tabs: { history: "Cronologia", map: "Mappa", codex: "Codice", inventory: "Inventario", stats: "Statistiche", craft: "Creazione", fronts: "Fronti", investigations: "Indagini", projects: "Progetti", achievements: "Traguardi", saves: "Salvataggi" },
} as const;
