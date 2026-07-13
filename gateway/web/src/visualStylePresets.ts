import type { VisualProfileUpdate } from "./types";

export type VisualStyleKey =
  | "auto"
  | "photorealistic"
  | "cinematic_fantasy"
  | "illustrated_fantasy"
  | "anime"
  | "custom";

export interface VisualStylePreset {
  key: VisualStyleKey;
  label: string;
  description: string;
  profile: VisualProfileUpdate;
}

const sharedNegative =
  "text, captions, logos, watermarks, UI, unreadable signage, generic AI composition, inconsistent character identity";

export const visualStylePresets: VisualStylePreset[] = [
  {
    key: "auto",
    label: "Auto",
    description: "Derives a coherent direction from the world's genre, tone, and known places.",
    profile: emptyProfile(),
  },
  {
    key: "photorealistic",
    label: "Photorealistic",
    description: "Real people and physically believable environments, photographed with natural texture and light.",
    profile: {
      world_style_prompt:
        "Photorealistic cinematic environment photography for a narrative world. Treat architecture, landscapes, props, weather, magic, and fantastical materials as physically real. Use natural or motivated lighting, believable shadows, grounded scale, real surface texture, restrained lens character, and subtle imperfections. The result should feel captured in a real place rather than rendered as concept art.",
      character_style_prompt:
        "Photorealistic real-human character photography. Preserve every canonical identity cue, expression, pose, hairstyle, hair color, outfit silhouette, accessories, palette, personality, and emotional tone. Translate stylized or fantastical features into plausible human anatomy with natural facial proportions, realistic eyes, visible skin texture and pores, subtle asymmetry, individual hair strands, believable fabric and materials, and a natural expression. Candid real-camera image with credible lighting.",
      negative_prompt: `${sharedNegative}, cosplay, wax figure, doll, 3D render, CGI, plastic skin, airbrushed beauty filter, heavily retouched studio portrait, generic AI face, altered outfit design, invented clothing, invented jewelry, invented accessories`,
      palette: "natural material colors, motivated practical light, restrained cinematic contrast",
    },
  },
  {
    key: "cinematic_fantasy",
    label: "Cinematic fantasy",
    description: "Grounded fantasy realism with dramatic scale, atmosphere, and film lighting.",
    profile: {
      world_style_prompt:
        "Grounded high-end cinematic fantasy. Build epic but physically plausible environments with weathered architecture, tactile terrain, coherent geography, volumetric atmosphere, motivated dramatic light, believable scale, and realistic materials. Magic should illuminate and affect the real scene rather than look like a flat graphic effect.",
      character_style_prompt:
        "Grounded cinematic fantasy characters with specific faces, credible anatomy, practical layered costumes, weathered materials, natural hair, and emotionally readable expressions. Preserve canonical identity, silhouette, palette, equipment, and design cues while making every element feel usable and physically present in the world.",
      negative_prompt: `${sharedNegative}, plastic armor, costume-shop clothing, weightless fabric, glossy game render, generic fantasy stock art, exaggerated anatomy`,
      palette: "deep natural colors, warm practical highlights, cool atmospheric shadows, cinematic contrast",
    },
  },
  {
    key: "illustrated_fantasy",
    label: "Illustrated fantasy",
    description: "Rich hand-painted storybook art with readable shapes and controlled detail.",
    profile: {
      world_style_prompt:
        "Premium hand-painted fantasy illustration for a narrative world. Use confident shapes, atmospheric depth, tactile brushwork, coherent architecture and terrain, elegant composition, readable silhouettes, and selective detail that supports the story rather than decorative noise.",
      character_style_prompt:
        "Expressive hand-painted fantasy character illustration. Preserve canonical identity, face, hairstyle, outfit silhouette, accessories, palette, and personality. Use believable anatomy, purposeful costume construction, nuanced expression, textured brushwork, and lighting coherent with the world.",
      negative_prompt: `${sharedNegative}, photobash seams, muddy values, flat clip art, chibi proportions, generic mobile-game character, excessive decorative detail`,
      palette: "story-led pigments, controlled saturation, warm focal accents, atmospheric shadow colors",
    },
  },
  {
    key: "anime",
    label: "Anime",
    description: "Polished anime key art that keeps identity, costume, and world continuity precise.",
    profile: {
      world_style_prompt:
        "Premium cinematic anime background art with coherent perspective, specific architecture, atmospheric depth, clean shape language, controlled detail, expressive weather, and lighting that supports the scene's emotion. Preserve the world's canonical geography, materials, and color identity.",
      character_style_prompt:
        "Premium anime character key art with consistent identity, clean anatomy, expressive but controlled facial acting, precise hair shape, faithful outfit silhouette, accessories, palette, and recognizable design cues. Use polished line control, dimensional cel shading, and lighting coherent with the scene.",
      negative_prompt: `${sharedNegative}, photorealism, live-action cosplay, chibi, super-deformed anatomy, inconsistent eye color, altered costume, extra accessories, flat unshaded draft`,
      palette: "clear color scripting, controlled saturation, cinematic light, distinct character accents",
    },
  },
  {
    key: "custom",
    label: "Custom prompt",
    description: "Write separate directions for the world and its characters.",
    profile: emptyProfile(),
  },
];

export function visualStylePreset(key: VisualStyleKey): VisualStylePreset {
  return visualStylePresets.find((preset) => preset.key === key) ?? visualStylePresets[0];
}

export function visualProfileForStyle(
  key: VisualStyleKey,
  custom: VisualProfileUpdate,
): VisualProfileUpdate {
  const source = key === "custom" ? custom : visualStylePreset(key).profile;
  return {
    world_style_prompt: source.world_style_prompt.trim(),
    character_style_prompt: source.character_style_prompt.trim(),
    negative_prompt: source.negative_prompt.trim(),
    palette: source.palette.trim(),
  };
}

export function emptyProfile(): VisualProfileUpdate {
  return {
    world_style_prompt: "",
    character_style_prompt: "",
    negative_prompt: "",
    palette: "",
  };
}
