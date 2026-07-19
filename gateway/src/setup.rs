//! Installation readiness is owned by the Go setup package. This adapter only
//! invokes its safe `oneday doctor --json` representation and narrows it for
//! authenticated browser use; it must not reproduce probe rules in Rust.

use crate::{engine, AppState};
use anyhow::{anyhow, Context};
use serde::{Deserialize, Serialize};
use std::sync::Arc;
use std::time::Duration;

const MAX_DOCTOR_JSON_BYTES: usize = 64 * 1024;
const MAX_PROBES: usize = 16;
const MAX_FIELD_BYTES: usize = 512;

#[derive(Debug, Clone, Deserialize)]
struct DoctorReport {
    probes: Vec<DoctorProbe>,
}

#[derive(Debug, Clone, Deserialize)]
struct DoctorProbe {
    name: String,
    code: String,
    status: String,
    required: bool,
    summary: String,
    #[serde(default)]
    action: String,
}

/// The browser-safe subset of the canonical `oneday doctor --json` report.
/// Config paths and process diagnostics remain server-only.
#[derive(Debug, Clone, Serialize)]
pub struct ReadinessReport {
    pub probes: Vec<ReadinessProbe>,
}

#[derive(Debug, Clone, Serialize)]
pub struct ReadinessProbe {
    pub name: String,
    pub code: String,
    pub status: String,
    pub required: bool,
    pub summary: String,
    pub action: String,
}

pub async fn readiness(state: Arc<AppState>) -> anyhow::Result<ReadinessReport> {
    let output =
        engine::run_gateway_command(&state, "doctor", &["--json"], None, Duration::from_secs(30))
            .await?;
    let report = parse_doctor_report(&output.stdout)?;

    // `oneday doctor` deliberately exits non-zero when a required probe fails.
    // A valid canonical report is still the expected readiness response.
    if !output.status.success() && report.probes.is_empty() {
        return Err(anyhow!("oneday doctor returned no readiness probes"));
    }
    Ok(report)
}

fn parse_doctor_report(raw: &[u8]) -> anyhow::Result<ReadinessReport> {
    if raw.len() > MAX_DOCTOR_JSON_BYTES {
        return Err(anyhow!(
            "oneday doctor readiness output exceeded {MAX_DOCTOR_JSON_BYTES} bytes"
        ));
    }
    let report = serde_json::from_slice::<DoctorReport>(raw)
        .context("decoding canonical oneday doctor readiness JSON")?;
    if report.probes.is_empty() || report.probes.len() > MAX_PROBES {
        return Err(anyhow!(
            "canonical doctor report has an invalid probe count"
        ));
    }

    let probes = report
        .probes
        .into_iter()
        .map(|probe| ReadinessProbe {
            name: bounded_text(probe.name),
            code: bounded_text(probe.code),
            status: bounded_text(probe.status),
            required: probe.required,
            summary: bounded_text(probe.summary),
            action: bounded_text(probe.action),
        })
        .collect::<Vec<_>>();
    if probes.iter().any(|probe| {
        probe.name.is_empty()
            || probe.code.is_empty()
            || probe.status.is_empty()
            || !valid_action(&probe.action)
    }) {
        return Err(anyhow!("canonical doctor report contains an invalid probe"));
    }
    Ok(ReadinessReport { probes })
}

fn valid_action(value: &str) -> bool {
    matches!(
        value,
        "" | "configure"
            | "check_credentials"
            | "check_connection"
            | "retry_later"
            | "check_capability"
            | "review_billing"
            | "create_backup"
            | "restore_empty_target"
            | "preserve_original"
    )
}

fn bounded_text(value: String) -> String {
    value
        .chars()
        .filter(|character| !character.is_control() || matches!(character, '\n' | '\t'))
        .take(MAX_FIELD_BYTES)
        .collect()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn preserves_required_and_optional_empty_install_probes_without_paths() {
        let report = parse_doctor_report(br#"{
          "config_source":"/private/config.yaml",
          "probes":[
            {"name":"narrative","code":"NARRATIVE_NOT_CONFIGURED","status":"failed","required":true,"summary":"no narrative provider is enabled","action":"configure"},
            {"name":"image","code":"IMAGE_DISABLED","status":"skipped","required":false,"summary":"image generation is disabled"},
            {"name":"tts","code":"TTS_DISABLED","status":"skipped","required":false,"summary":"text-to-speech is disabled"}
          ]
        }"#).expect("empty-install report parses");

        assert_eq!(report.probes.len(), 3);
        assert!(report.probes[0].required);
        assert_eq!(report.probes[0].status, "failed");
        assert!(!report.probes[1].required);
        assert_eq!(
            serde_json::to_string(&report).unwrap(),
            r#"{"probes":[{"name":"narrative","code":"NARRATIVE_NOT_CONFIGURED","status":"failed","required":true,"summary":"no narrative provider is enabled","action":"configure"},{"name":"image","code":"IMAGE_DISABLED","status":"skipped","required":false,"summary":"image generation is disabled","action":""},{"name":"tts","code":"TTS_DISABLED","status":"skipped","required":false,"summary":"text-to-speech is disabled","action":""}]}"#
        );
    }

    #[test]
    fn preserves_configured_readiness_state() {
        let report = parse_doctor_report(br#"{
          "probes":[
            {"name":"narrative","code":"NARRATIVE_READY","status":"ready","required":true,"summary":"narrative provider is ready"},
            {"name":"image","code":"IMAGE_READY","status":"ready","required":false,"summary":"image generation is configured"},
            {"name":"tts","code":"TTS_READY","status":"ready","required":false,"summary":"text-to-speech is configured"}
          ]
        }"#).expect("configured report parses");

        assert!(report.probes.iter().all(|probe| probe.status == "ready"));
        assert!(report
            .probes
            .iter()
            .any(|probe| probe.name == "tts" && !probe.required));
    }

    #[test]
    fn rejects_unbounded_or_missing_probe_output() {
        assert!(parse_doctor_report(br#"{"probes":[]}"#).is_err());
        assert!(parse_doctor_report(&vec![b'x'; MAX_DOCTOR_JSON_BYTES + 1]).is_err());
        assert!(parse_doctor_report(br#"{"probes":[{"name":"narrative","code":"NARRATIVE_TIMEOUT","status":"failed","required":true,"summary":"safe","action":"untrusted_action"}]}"#).is_err());
    }
}
