use axum::http::StatusCode;

#[derive(Clone, Copy, Debug, Eq, PartialEq)]
pub enum PublicErrorKind {
    BadRequest,
    Conflict,
    NotFound,
    PayloadTooLarge,
    UnsupportedMediaType,
    UnprocessableEntity,
}

impl PublicErrorKind {
    pub fn status(self) -> StatusCode {
        match self {
            Self::BadRequest => StatusCode::BAD_REQUEST,
            Self::Conflict => StatusCode::CONFLICT,
            Self::NotFound => StatusCode::NOT_FOUND,
            Self::PayloadTooLarge => StatusCode::PAYLOAD_TOO_LARGE,
            Self::UnsupportedMediaType => StatusCode::UNSUPPORTED_MEDIA_TYPE,
            Self::UnprocessableEntity => StatusCode::UNPROCESSABLE_ENTITY,
        }
    }
}

#[derive(Debug, thiserror::Error)]
#[error("{message}")]
pub struct PublicError {
    pub kind: PublicErrorKind,
    pub code: &'static str,
    pub message: String,
}

impl PublicError {
    pub fn bad_request(code: &'static str, message: impl Into<String>) -> Self {
        Self {
            kind: PublicErrorKind::BadRequest,
            code,
            message: message.into(),
        }
    }

    pub fn conflict(code: &'static str, message: impl Into<String>) -> Self {
        Self {
            kind: PublicErrorKind::Conflict,
            code,
            message: message.into(),
        }
    }

    pub fn not_found(code: &'static str, message: impl Into<String>) -> Self {
        Self {
            kind: PublicErrorKind::NotFound,
            code,
            message: message.into(),
        }
    }

    pub fn payload_too_large(code: &'static str, message: impl Into<String>) -> Self {
        Self {
            kind: PublicErrorKind::PayloadTooLarge,
            code,
            message: message.into(),
        }
    }

    pub fn unsupported_media_type(code: &'static str, message: impl Into<String>) -> Self {
        Self {
            kind: PublicErrorKind::UnsupportedMediaType,
            code,
            message: message.into(),
        }
    }

    pub fn unprocessable_entity(code: &'static str, message: impl Into<String>) -> Self {
        Self {
            kind: PublicErrorKind::UnprocessableEntity,
            code,
            message: message.into(),
        }
    }
}
