use anyhow::{anyhow, Context};
use clap::Parser;
use serde::{Deserialize, Serialize};
use std::path::{Path, PathBuf};

#[derive(Debug, Parser)]
#[command(name = "oneday-gateway")]
pub struct Args {
    #[arg(long, env = "ONEDAY_GATEWAY_ADDR", default_value = "0.0.0.0:8788")]
    pub addr: String,

    #[arg(long, env = "ONEDAY_ROOT", default_value = ".")]
    pub oneday_root: PathBuf,

    #[arg(long, env = "ONEDAY_CONFIG")]
    pub config_path: Option<PathBuf>,

    #[arg(long, env = "ONEDAY_DB_PATH")]
    pub db_path: Option<PathBuf>,

    #[arg(long, env = "ONEDAY_BIN")]
    pub oneday_bin: Option<PathBuf>,

    #[arg(long, env = "ONEDAY_GATEWAY_STATIC_DIR")]
    pub static_dir: Option<PathBuf>,
}

impl Args {
    pub fn parse_args() -> Self {
        Self::parse()
    }
}

#[derive(Clone, Debug)]
pub struct ResolvedPaths {
    pub oneday_root: PathBuf,
    pub config_path: PathBuf,
    pub db_path: PathBuf,
    pub oneday_bin: PathBuf,
    pub static_dir: PathBuf,
}

#[derive(Debug, Deserialize)]
struct OneDayConfig {
    data_dir: Option<String>,
    ai: Option<AIConfig>,
}

#[derive(Debug, Deserialize)]
struct AIConfig {
    provider_priority: Option<Vec<String>>,
    codex: Option<CodexConfig>,
    litellm: Option<OpenAICompatConfig>,
    openrouter: Option<OpenAICompatConfig>,
    embedding: Option<EmbeddingConfig>,
    ascii_art: Option<AsciiArtConfig>,
    generation: Option<GenerationConfig>,
}

#[derive(Debug, Deserialize)]
struct CodexConfig {
    model: Option<String>,
    reasoning: Option<String>,
}

#[derive(Debug, Deserialize)]
struct OpenAICompatConfig {
    default_model: Option<String>,
}

#[derive(Debug, Deserialize)]
struct EmbeddingConfig {
    model: Option<String>,
    provider: Option<String>,
}

#[derive(Debug, Deserialize)]
struct AsciiArtConfig {
    model: Option<String>,
}

#[derive(Debug, Deserialize)]
struct GenerationConfig {
    utility_model: Option<String>,
    repair_model: Option<String>,
    repair_fallback_models: Option<Vec<String>>,
}

#[derive(Clone, Debug, Serialize)]
pub struct ModelSettings {
    pub provider_priority: Vec<String>,
    pub narrative_models: Vec<String>,
    pub utility_models: Vec<String>,
    pub repair_models: Vec<String>,
    pub image_models: Vec<String>,
    pub embedding_model: String,
    pub embedding_provider: String,
    pub codex_reasoning: String,
    pub tts_status: String,
}

pub fn resolve_paths(args: &Args) -> anyhow::Result<ResolvedPaths> {
    let oneday_root = absolute_path(&args.oneday_root).context("resolving oneday root")?;
    let config_path = args
        .config_path
        .clone()
        .unwrap_or_else(|| oneday_root.join("config.yaml"));
    let config_path = absolute_path_relative(&oneday_root, &config_path)?;
    let oneday_bin = args
        .oneday_bin
        .clone()
        .unwrap_or_else(|| oneday_root.join("oneday"));
    let oneday_bin = absolute_path_relative(&oneday_root, &oneday_bin)?;
    let static_dir = args
        .static_dir
        .clone()
        .unwrap_or_else(|| oneday_root.join("gateway/web/dist"));
    let static_dir = absolute_path_relative(&oneday_root, &static_dir)?;

    let db_path = if let Some(path) = &args.db_path {
        absolute_path_relative(&oneday_root, path)?
    } else {
        let data_dir = read_data_dir(&config_path).unwrap_or_else(|_| PathBuf::from("oneday_data"));
        let data_dir = absolute_path_relative(&oneday_root, &data_dir)?;
        data_dir.join("oneday.db")
    };

    if !db_path.exists() {
        return Err(anyhow!("database does not exist: {}", db_path.display()));
    }
    if !oneday_bin.exists() {
        return Err(anyhow!(
            "oneday binary does not exist: {}",
            oneday_bin.display()
        ));
    }
    if !static_dir.join("index.html").exists() {
        return Err(anyhow!(
            "gateway static index does not exist: {}",
            static_dir.join("index.html").display()
        ));
    }

    Ok(ResolvedPaths {
        oneday_root,
        config_path,
        db_path,
        oneday_bin,
        static_dir,
    })
}

fn read_data_dir(config_path: &Path) -> anyhow::Result<PathBuf> {
    let raw = std::fs::read_to_string(config_path)
        .with_context(|| format!("reading {}", config_path.display()))?;
    let cfg: OneDayConfig =
        serde_yaml::from_str(&raw).with_context(|| format!("parsing {}", config_path.display()))?;
    Ok(PathBuf::from(
        cfg.data_dir.unwrap_or_else(|| "./oneday_data".to_string()),
    ))
}

pub fn read_model_settings(config_path: &Path) -> anyhow::Result<ModelSettings> {
    let raw = std::fs::read_to_string(config_path)
        .with_context(|| format!("reading {}", config_path.display()))?;
    let cfg: OneDayConfig =
        serde_yaml::from_str(&raw).with_context(|| format!("parsing {}", config_path.display()))?;
    let ai = cfg.ai;
    let provider_priority = ai
        .as_ref()
        .and_then(|value| value.provider_priority.clone())
        .unwrap_or_default();
    let mut narrative_models = Vec::new();
    push_opt(
        &mut narrative_models,
        ai.as_ref()
            .and_then(|value| value.litellm.as_ref())
            .and_then(|value| value.default_model.clone()),
    );
    push_opt(
        &mut narrative_models,
        ai.as_ref()
            .and_then(|value| value.openrouter.as_ref())
            .and_then(|value| value.default_model.clone()),
    );
    push_opt(
        &mut narrative_models,
        ai.as_ref()
            .and_then(|value| value.codex.as_ref())
            .and_then(|value| value.model.clone()),
    );

    let mut utility_models = Vec::new();
    push_opt(
        &mut utility_models,
        ai.as_ref()
            .and_then(|value| value.generation.as_ref())
            .and_then(|value| value.utility_model.clone()),
    );

    let mut repair_models = Vec::new();
    push_opt(
        &mut repair_models,
        ai.as_ref()
            .and_then(|value| value.generation.as_ref())
            .and_then(|value| value.repair_model.clone()),
    );
    if let Some(fallbacks) = ai
        .as_ref()
        .and_then(|value| value.generation.as_ref())
        .and_then(|value| value.repair_fallback_models.clone())
    {
        for model in fallbacks {
            push_opt(&mut repair_models, Some(model));
        }
    }

    let mut image_models = Vec::new();
    push_opt(
        &mut image_models,
        ai.as_ref()
            .and_then(|value| value.ascii_art.as_ref())
            .and_then(|value| value.model.clone()),
    );

    let embedding_model = ai
        .as_ref()
        .and_then(|value| value.embedding.as_ref())
        .and_then(|value| value.model.clone())
        .unwrap_or_default();
    let embedding_provider = ai
        .as_ref()
        .and_then(|value| value.embedding.as_ref())
        .and_then(|value| value.provider.clone())
        .unwrap_or_else(|| "auto".to_string());
    let codex_reasoning = ai
        .as_ref()
        .and_then(|value| value.codex.as_ref())
        .and_then(|value| value.reasoning.clone())
        .unwrap_or_else(|| "off".to_string());

    Ok(ModelSettings {
        provider_priority,
        narrative_models,
        utility_models,
        repair_models,
        image_models,
        embedding_model,
        embedding_provider,
        codex_reasoning,
        tts_status: "planned".to_string(),
    })
}

fn push_opt(values: &mut Vec<String>, value: Option<String>) {
    let Some(value) = value else {
        return;
    };
    let clean = value.trim();
    if clean.is_empty() || values.iter().any(|item| item == clean) {
        return;
    }
    values.push(clean.to_string());
}

fn absolute_path(path: &Path) -> anyhow::Result<PathBuf> {
    if path.is_absolute() {
        Ok(path.to_path_buf())
    } else {
        Ok(std::env::current_dir()?.join(path))
    }
}

fn absolute_path_relative(root: &Path, path: &Path) -> anyhow::Result<PathBuf> {
    if path.is_absolute() {
        Ok(path.to_path_buf())
    } else {
        Ok(root.join(path))
    }
}
