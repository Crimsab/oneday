use std::fmt;

/// A process-only value. It deliberately cannot be serialized, cloned, or
/// formatted, which keeps launch secrets out of settings files and logs.
pub struct LaunchSecret([u8; 32]);

impl LaunchSecret {
    pub fn generate() -> Result<Self, String> {
        let mut bytes = [0_u8; 32];
        getrandom::fill(&mut bytes)
            .map_err(|error| format!("Could not create a local launch secret: {error}"))?;
        Ok(Self(bytes))
    }

    pub fn environment_value(&self) -> String {
        hex(&self.0)
    }
}

impl fmt::Debug for LaunchSecret {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        formatter.write_str("LaunchSecret([redacted])")
    }
}

pub fn profile_id() -> Result<String, String> {
    let mut bytes = [0_u8; 16];
    getrandom::fill(&mut bytes)
        .map_err(|error| format!("Could not create a standalone profile ID: {error}"))?;
    Ok(hex(&bytes))
}

fn hex(bytes: &[u8]) -> String {
    const DIGITS: &[u8; 16] = b"0123456789abcdef";
    let mut encoded = String::with_capacity(bytes.len() * 2);
    for byte in bytes {
        encoded.push(DIGITS[(byte >> 4) as usize] as char);
        encoded.push(DIGITS[(byte & 0x0f) as usize] as char);
    }
    encoded
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn launch_secrets_are_fixed_length_hex_without_debug_contents() {
        let secret = LaunchSecret::generate().expect("secret");
        assert_eq!(secret.environment_value().len(), 64);
        assert_eq!(format!("{secret:?}"), "LaunchSecret([redacted])");
    }

    #[test]
    fn standalone_profile_ids_are_opaque() {
        let id = profile_id().expect("profile id");
        assert_eq!(id.len(), 32);
        assert!(id.bytes().all(|byte| byte.is_ascii_hexdigit()));
    }
}
