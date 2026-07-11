#![cfg_attr(not(test), allow(dead_code))]

use serde::{Deserialize, Serialize};

#[derive(Clone, Debug, Deserialize, Serialize)]
pub struct ChallengeFixture {
    pub instance: ChallengeInstance,
    pub input: ChallengeInput,
    pub resolution: ChallengeResolution,
}

#[derive(Clone, Debug, Deserialize, Serialize)]
pub struct ChallengeInstance {
    pub protocol_version: u32,
    pub id: String,
    pub story_id: Option<String>,
    pub branch_id: Option<String>,
    pub turn: i64,
    pub definition: ChallengeDefinition,
    pub seed: i64,
    pub policy: OutcomePolicy,
    pub timing: Option<serde_json::Value>,
}

#[derive(Clone, Debug, Deserialize, Serialize)]
pub struct ChallengeDefinition {
    pub id: String,
    pub kind: String,
    pub description: Option<String>,
    pub difficulty: i64,
}

#[derive(Clone, Debug, Deserialize, Serialize)]
pub struct OutcomePolicy {
    pub id: String,
    pub difficulty_profile: Option<String>,
    pub consequence_budget: i64,
    pub critical_band: i64,
    pub fairness: String,
}

#[derive(Clone, Debug, Deserialize, Serialize)]
pub struct ChallengeInput {
    pub actor_id: Option<String>,
    pub intent: String,
    #[serde(default)]
    pub modifiers: Vec<ChallengeModifier>,
}

#[derive(Clone, Debug, Deserialize, Serialize)]
pub struct ChallengeModifier {
    pub source: String,
    pub value: i64,
}

#[derive(Clone, Debug, Deserialize, Serialize)]
pub struct ChallengeResolution {
    pub protocol_version: u32,
    pub instance_id: String,
    pub input: ChallengeInput,
    pub outcome: OutcomeEnvelope,
}

#[derive(Clone, Debug, Deserialize, Serialize)]
pub struct OutcomeEnvelope {
    pub version: u32,
    pub degree: String,
    pub difficulty: i64,
    pub seed: i64,
    pub roll: i64,
    #[serde(default)]
    pub modifiers: Vec<ChallengeModifier>,
    pub total: i64,
    pub margin: i64,
    #[serde(default)]
    pub costs: Vec<serde_json::Value>,
    #[serde(default)]
    pub consequences: Vec<String>,
    #[serde(default)]
    pub state_deltas: Vec<serde_json::Value>,
    #[serde(default)]
    pub revealed_facts: Vec<String>,
    #[serde(default)]
    pub follow_up_pressure: i64,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn canonical_fixture_deserializes_without_loss() {
        let fixture: ChallengeFixture =
            serde_json::from_str(include_str!("../../contracts/challenge-v1.json")).unwrap();
        assert_eq!(fixture.instance.protocol_version, 1);
        assert_eq!(fixture.instance.id, fixture.resolution.instance_id);
        assert_eq!(fixture.instance.seed, fixture.resolution.outcome.seed);
        assert_eq!(fixture.input.intent, fixture.resolution.input.intent);
    }
}
