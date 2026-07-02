use anyhow::{anyhow, Context};
use clap::Parser;
use serde::Deserialize;
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
}

#[derive(Debug, Deserialize)]
struct OneDayConfig {
    data_dir: Option<String>,
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

    Ok(ResolvedPaths {
        oneday_root,
        config_path,
        db_path,
        oneday_bin,
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
