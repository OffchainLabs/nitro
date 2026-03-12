// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use std::sync::Arc;

use alloy_rpc_types_engine::JwtSecret;
use anyhow::Context;
use axum::{
    extract::State,
    http::{Request, StatusCode},
    middleware::Next,
    response::Response,
};
use axum_extra::{
    headers::{authorization::Bearer, Authorization},
    TypedHeader,
};

use crate::config::ServerState;

/// Loads a JWT secret from a hex string or a path to a file containing one.
///
/// Accepts:
/// - A 64-char hex string (with or without `0x` prefix)
/// - A path to a file whose contents are such a string
///
/// Matches the format used by Go's `signature.LoadSigningKey`.
pub fn load_secret(value: &str) -> anyhow::Result<JwtSecret> {
    if looks_like_hex(value) {
        JwtSecret::from_hex(value).map_err(|e| anyhow::anyhow!("{e}"))
    } else {
        let hex_str = std::fs::read_to_string(value)
            .with_context(|| format!("failed to read JWT secret file {value:?}"))?;
        JwtSecret::from_hex(hex_str.trim()).map_err(|e| anyhow::anyhow!("{e}"))
    }
}

/// Returns true if the value looks like a 64-char hex string (with or without `0x`).
fn looks_like_hex(s: &str) -> bool {
    let s = s.strip_prefix("0x").unwrap_or(s);
    s.len() == 64 && s.chars().all(|c| c.is_ascii_hexdigit())
}

/// Axum middleware that enforces JWT authentication when the server has a secret
/// configured, and is a no-op otherwise.
pub async fn auth_middleware(
    State(state): State<Arc<ServerState>>,
    bearer: Option<TypedHeader<Authorization<Bearer>>>,
    request: Request<axum::body::Body>,
    next: Next,
) -> Result<Response, StatusCode> {
    let Some(secret) = &state.jwt_secret else {
        return Ok(next.run(request).await);
    };

    let TypedHeader(Authorization(bearer)) = bearer.ok_or(StatusCode::UNAUTHORIZED)?;

    secret.validate(bearer.token()).map_err(|err| {
        tracing::warn!("JWT validation failed: {err}");
        StatusCode::UNAUTHORIZED
    })?;

    Ok(next.run(request).await)
}

#[cfg(test)]
mod tests {
    use super::*;

    fn secret() -> String {
        "deadbeef".repeat(8) // 64 chars
    }

    #[test]
    fn load_secret_from_hex_string() {
        let secret = load_secret(&secret()).unwrap();
        assert_eq!(secret.as_bytes()[0], 0xde);
        assert_eq!(secret.as_bytes()[1], 0xad);
    }

    #[test]
    fn load_secret_from_hex_string_with_prefix() {
        let hex = format!("0x{}", secret());
        let secret = load_secret(&hex).unwrap();
        assert_eq!(secret.as_bytes()[0], 0xde);
    }

    #[test]
    fn load_secret_from_file() {
        let dir = tempdir::TempDir::new("jwt_test").unwrap();
        let path = dir.path().join("jwt.hex");
        std::fs::write(&path, secret()).unwrap();
        let secret = load_secret(path.to_str().unwrap()).unwrap();
        assert_eq!(secret.as_bytes()[0], 0xde);
    }

    #[test]
    fn load_secret_wrong_length_errors() {
        assert!(load_secret("deadbeef").is_err()); // too short
    }
}
