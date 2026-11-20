use std::fmt;
use std::time::Duration;

use serde::Deserialize;

use crate::client::{BoxError, HttpResponse};

/// What a call answers instead of its resource.
#[derive(Debug)]
#[non_exhaustive]
pub enum Error {
    /// The API refused the request.
    Api(ApiError),
    /// The request did not reach the API, or its answer did not arrive.
    Transport(BoxError),
    /// A body could not be written as JSON, or the API answered something
    /// this SDK does not read.
    Json(serde_json::Error),
    /// The client's base URL is not https and not a loopback address, so no
    /// request was sent: the API key would cross the network in clear text.
    InsecureBaseUrl,
    /// No API key was given and `MEENG_API_KEY` is not set.
    MissingApiKey,
}

/// A request the API refused.
///
/// `code` is the API's, such as `NOT_UNIQUE`, `FORBIDDEN` or `NOT_FOUND`; it is
/// `INTERNAL_SERVER_ERROR` when the answer carried none. `retry_after` is how
/// long to wait on a 429.
#[derive(Debug, Clone, PartialEq, Eq)]
#[non_exhaustive]
pub struct ApiError {
    /// The HTTP status.
    pub status: u16,
    /// The API's code for the refusal.
    pub code: String,
    /// What went wrong, for a person to read.
    pub message: String,
    /// How long to wait before trying again.
    pub retry_after: Option<Duration>,
}

#[derive(Deserialize, Default)]
struct Problem {
    code: Option<String>,
    message: Option<String>,
}

/// `{"error": {"code", "message"}}`, `{"error": "text"}` or the error itself.
#[derive(Deserialize, Default)]
struct ErrorBody {
    error: Option<serde_json::Value>,
    code: Option<String>,
    message: Option<String>,
}

impl ApiError {
    /// The error a refused response carries. A refusal without a JSON body is
    /// still a refusal.
    pub(crate) fn from_response(response: &HttpResponse) -> Self {
        let body: ErrorBody = serde_json::from_slice(&response.body).unwrap_or_default();
        let problem = match body.error {
            Some(serde_json::Value::String(text)) => Problem {
                code: None,
                message: Some(text),
            },
            Some(other) => serde_json::from_value(other).unwrap_or_default(),
            None => Problem {
                code: body.code,
                message: body.message,
            },
        };
        Self {
            status: response.status,
            code: problem
                .code
                .filter(|code| !code.is_empty())
                .unwrap_or_else(|| "INTERNAL_SERVER_ERROR".to_owned()),
            message: problem
                .message
                .filter(|message| !message.is_empty())
                .unwrap_or_else(|| reason(response.status).to_owned()),
            retry_after: response
                .header("retry-after")
                .and_then(|value| value.trim().parse::<u64>().ok())
                .map(Duration::from_secs),
        }
    }
}

fn reason(status: u16) -> &'static str {
    match status {
        400 => "Bad Request",
        401 => "Unauthorized",
        403 => "Forbidden",
        404 => "Not Found",
        409 => "Conflict",
        422 => "Unprocessable Entity",
        429 => "Too Many Requests",
        500 => "Internal Server Error",
        502 => "Bad Gateway",
        503 => "Service Unavailable",
        _ => "Error",
    }
}

impl fmt::Display for ApiError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "meeng: {} {}: {}", self.status, self.code, self.message)
    }
}

impl std::error::Error for ApiError {}

impl fmt::Display for Error {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Error::Api(error) => error.fmt(f),
            Error::Transport(error) => write!(f, "meeng: {error}"),
            Error::Json(error) => write!(f, "meeng: json: {error}"),
            Error::InsecureBaseUrl => f.write_str("meeng: the base URL must be https (http is accepted only for localhost, 127.0.0.0/8 and ::1), so the API key is never sent in clear text"),
            Error::MissingApiKey => f.write_str("meeng: missing API key: pass it to Client::new or set MEENG_API_KEY"),
        }
    }
}

impl std::error::Error for Error {
    fn source(&self) -> Option<&(dyn std::error::Error + 'static)> {
        match self {
            Error::Api(error) => Some(error),
            Error::Transport(error) => Some(error.as_ref()),
            Error::Json(error) => Some(error),
            Error::InsecureBaseUrl | Error::MissingApiKey => None,
        }
    }
}

impl From<ApiError> for Error {
    fn from(error: ApiError) -> Self {
        Error::Api(error)
    }
}

impl Error {
    /// The refusal, when the API refused the request.
    pub fn api(&self) -> Option<&ApiError> {
        match self {
            Error::Api(error) => Some(error),
            _ => None,
        }
    }
}
