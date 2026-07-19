use serde::Serialize;

#[derive(Clone, Debug, PartialEq, Eq, Serialize)]
#[serde(rename_all = "camelCase", tag = "state")]
pub enum Lifecycle {
    Stopped,
    Starting,
    Ready { endpoint: String },
    Draining,
    Failed { message: String },
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn lifecycle_serializes_without_internal_process_details() {
        let value = serde_json::to_value(Lifecycle::Ready {
            endpoint: "http://127.0.0.1:49152/".into(),
        })
        .expect("serialize lifecycle");
        assert_eq!(value["state"], "ready");
        assert!(value.get("pid").is_none());
    }
}
