use std::fmt;
use std::sync::Arc;
use std::time::Duration;

use serde::de::DeserializeOwned;
use serde::Serialize;
use ureq::RequestExt;

use crate::error::{ApiError, Error};
use crate::version::{API_PREFIX, DEFAULT_BASE_URL, VERSION};

/// Any error a [`Transport`] answers.
pub type BoxError = Box<dyn std::error::Error + Send + Sync>;

/// One request, as a [`Transport`] sends it.
#[derive(Debug, Clone, PartialEq, Eq)]
#[non_exhaustive]
pub struct HttpRequest {
    /// GET, POST…
    pub method: String,
    /// The whole address, with the query.
    pub url: String,
    /// The headers, in the order they are sent.
    pub headers: Vec<(String, String)>,
    /// The JSON body of a write, if it has one.
    pub body: Option<Vec<u8>>,
    /// How long the call may take.
    pub timeout: Duration,
}

impl HttpRequest {
    /// The first value of a header, its name in any case.
    pub fn header(&self, name: &str) -> Option<&str> {
        find(&self.headers, name)
    }
}

/// One answer, as a [`Transport`] reads it.
#[derive(Debug, Clone, PartialEq, Eq, Default)]
pub struct HttpResponse {
    /// The HTTP status.
    pub status: u16,
    /// The headers.
    pub headers: Vec<(String, String)>,
    /// The body.
    pub body: Vec<u8>,
}

impl HttpResponse {
    /// An answer with this status and body and no headers.
    pub fn new(status: u16, body: impl Into<Vec<u8>>) -> Self {
        Self {
            status,
            headers: Vec::new(),
            body: body.into(),
        }
    }

    /// The same answer with one more header.
    pub fn with_header(mut self, name: impl Into<String>, value: impl Into<String>) -> Self {
        self.headers.push((name.into(), value.into()));
        self
    }

    /// The first value of a header, its name in any case.
    pub fn header(&self, name: &str) -> Option<&str> {
        find(&self.headers, name)
    }
}

fn find<'a>(headers: &'a [(String, String)], name: &str) -> Option<&'a str> {
    headers
        .iter()
        .find(|(key, _)| key.eq_ignore_ascii_case(name))
        .map(|(_, value)| value.as_str())
}

/// Sends a request and answers what came back, whatever its status. The SDK
/// sends through [`UreqTransport`]; a test, or a program with its own HTTP
/// client, gives another one to [`ClientBuilder::transport`]. A function
/// from [`HttpRequest`] to `Result<HttpResponse, BoxError>` is one.
pub trait Transport: Send + Sync {
    /// Sends one request.
    fn send(&self, request: HttpRequest) -> Result<HttpResponse, BoxError>;
}

impl<F> Transport for F
where
    F: Fn(HttpRequest) -> Result<HttpResponse, BoxError> + Send + Sync,
{
    fn send(&self, request: HttpRequest) -> Result<HttpResponse, BoxError> {
        self(request)
    }
}

/// The HTTP the SDK sends through: [ureq](https://docs.rs/ureq), blocking,
/// with rustls.
#[derive(Debug, Clone)]
pub struct UreqTransport {
    agent: ureq::Agent,
}

impl UreqTransport {
    /// A transport with its own connection pool.
    pub fn new() -> Self {
        let agent = ureq::Agent::config_builder()
            .http_status_as_error(false)
            // The API never redirects, and a redirect would carry the key along.
            .max_redirects(0)
            .max_redirects_will_error(false)
            .build()
            .new_agent();
        Self { agent }
    }
}

impl Default for UreqTransport {
    fn default() -> Self {
        Self::new()
    }
}

impl Transport for UreqTransport {
    fn send(&self, request: HttpRequest) -> Result<HttpResponse, BoxError> {
        let mut builder = ureq::http::Request::builder()
            .method(request.method.as_str())
            .uri(request.url.as_str());
        for (name, value) in &request.headers {
            builder = builder.header(name.as_str(), value.as_str());
        }
        let timeout = Some(request.timeout);
        let mut response = match request.body {
            Some(body) => builder
                .body(body)?
                .with_agent(&self.agent)
                .configure()
                .timeout_global(timeout)
                .run()?,
            None => builder
                .body(())?
                .with_agent(&self.agent)
                .configure()
                .timeout_global(timeout)
                .run()?,
        };
        let headers = response
            .headers()
            .iter()
            .filter_map(|(name, value)| {
                Some((name.as_str().to_owned(), value.to_str().ok()?.to_owned()))
            })
            .collect();
        let status = response.status().as_u16();
        let body = response.body_mut().read_to_vec()?;
        Ok(HttpResponse {
            status,
            headers,
            body,
        })
    }
}

/// Options of one call: the key that makes a send safe to retry.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct RequestOptions {
    /// Makes a send safe to retry: the same key with the same body answers the
    /// first result again; with another body, a 409 `NOT_UNIQUE`. The key is up
    /// to 256 characters and expires after 24 hours. The API reads it on
    /// `emails().send` and `emails().batch`.
    pub idempotency_key: Option<String>,
}

/// meeng's API with one team's API key. [`Client::from_env`] reads the key from
/// `MEENG_API_KEY`, and the address of the app from `MEENG_BASE_URL`.
///
/// A client is cheap to clone, and its clones share one connection pool.
///
/// A call that may be repeated is repeated up to
/// [`max_retries`](ClientBuilder::max_retries) times (2 by default; 0 sends
/// every call once) after the first attempt. It is repeated when the API
/// answers 429, which means it was not processed, so any call may be; and when
/// the API answers 500, 502, 503 or 504, or the connection fails, only if
/// repeating it cannot do the work twice: a read, a PUT, a DELETE, or any call
/// that carries an idempotency key. Any other refusal, a 409 included, is never
/// repeated. The first repetition waits
/// [`retry_delay`](ClientBuilder::retry_delay) (500 ms by default), each next
/// one twice as long, and a Retry-After from the API replaces the wait.
#[derive(Clone)]
pub struct Client {
    api_key: Option<String>,
    base_url: String,
    timeout: Duration,
    max_retries: u32,
    retry_delay: Duration,
    transport: Arc<dyn Transport>,
}

/// The longest a call waits before a repetition.
pub const MAX_RETRY_DELAY: Duration = Duration::from_secs(30);

impl fmt::Debug for Client {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.debug_struct("Client")
            .field("base_url", &self.base_url)
            .field("timeout", &self.timeout)
            .finish_non_exhaustive()
    }
}

/// Configures a [`Client`].
pub struct ClientBuilder {
    api_key: Option<String>,
    base_url: String,
    timeout: Duration,
    max_retries: u32,
    retry_delay: Duration,
    transport: Option<Arc<dyn Transport>>,
}

impl fmt::Debug for ClientBuilder {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.debug_struct("ClientBuilder")
            .field("base_url", &self.base_url)
            .field("timeout", &self.timeout)
            .finish_non_exhaustive()
    }
}

impl ClientBuilder {
    /// Points the client at another app, such as a self-hosted one: the address
    /// of the app, without `/api/v1`. It must be https; http is accepted only
    /// for localhost, 127.0.0.0/8 and ::1, and any other makes every call
    /// answer [`Error::InsecureBaseUrl`] before anything is sent.
    pub fn base_url(mut self, base_url: impl Into<String>) -> Self {
        self.base_url = base_url.into().trim_end_matches('/').to_owned();
        self
    }

    /// How long an attempt may take; 30 seconds by default.
    pub fn timeout(mut self, timeout: Duration) -> Self {
        self.timeout = timeout;
        self
    }

    /// How many times a call that may be repeated is repeated after the first
    /// attempt; 2 by default, and 0 sends every call once.
    pub fn max_retries(mut self, max_retries: u32) -> Self {
        self.max_retries = max_retries;
        self
    }

    /// The wait before the first repetition; 500 ms by default. Each next
    /// repetition waits twice as long, and a Retry-After the API answers
    /// replaces the wait.
    pub fn retry_delay(mut self, retry_delay: Duration) -> Self {
        self.retry_delay = retry_delay;
        self
    }

    /// Sends the requests through `transport`.
    pub fn transport(mut self, transport: impl Transport + 'static) -> Self {
        self.transport = Some(Arc::new(transport));
        self
    }

    /// The client.
    pub fn build(self) -> Client {
        Client {
            api_key: self.api_key,
            base_url: self.base_url,
            timeout: self.timeout,
            max_retries: self.max_retries,
            retry_delay: self.retry_delay,
            transport: self
                .transport
                .unwrap_or_else(|| Arc::new(UreqTransport::new())),
        }
    }
}

impl Client {
    /// A client for the team `api_key` belongs to. The address of the app is
    /// `MEENG_BASE_URL` when that is set.
    pub fn new(api_key: impl Into<String>) -> Self {
        Self::builder(api_key).build()
    }

    /// A client for the key in `MEENG_API_KEY`; [`Error::MissingApiKey`] when
    /// it is not set.
    pub fn from_env() -> Result<Self, Error> {
        match std::env::var("MEENG_API_KEY") {
            Ok(key) if !key.is_empty() => Ok(Self::new(key)),
            _ => Err(Error::MissingApiKey),
        }
    }

    /// Configures a client for the team `api_key` belongs to.
    pub fn builder(api_key: impl Into<String>) -> ClientBuilder {
        let base_url = std::env::var("MEENG_BASE_URL")
            .ok()
            .filter(|value| !value.is_empty())
            .unwrap_or_else(|| DEFAULT_BASE_URL.to_owned());
        ClientBuilder {
            api_key: Some(api_key.into()).filter(|key| !key.is_empty()),
            base_url: base_url.trim_end_matches('/').to_owned(),
            timeout: Duration::from_secs(30),
            max_retries: 2,
            retry_delay: Duration::from_millis(500),
            transport: None,
        }
    }

    /// The address every call goes to, with the API prefix.
    pub fn url(&self) -> String {
        format!("{}{}", self.base_url, API_PREFIX)
    }

    /// Calls the API and reads its answer; a refusal is [`Error::Api`].
    pub(crate) fn request<T, B>(
        &self,
        method: &str,
        path: &str,
        query: &[(String, String)],
        body: Option<&B>,
        options: &RequestOptions,
    ) -> Result<T, Error>
    where
        T: DeserializeOwned,
        B: Serialize + ?Sized,
    {
        let Some(api_key) = &self.api_key else {
            return Err(Error::MissingApiKey);
        };
        if !secure_base_url(&self.base_url) {
            return Err(Error::InsecureBaseUrl);
        }
        let mut url = format!("{}{}{}", self.base_url, API_PREFIX, path);
        let pairs: Vec<String> = query
            .iter()
            .filter(|(_, value)| !value.is_empty())
            .map(|(name, value)| format!("{}={}", escape(name), escape(value)))
            .collect();
        if !pairs.is_empty() {
            url.push('?');
            url.push_str(&pairs.join("&"));
        }
        let mut headers = vec![
            ("Authorization".to_owned(), format!("Bearer {api_key}")),
            ("Accept".to_owned(), "application/json".to_owned()),
            ("User-Agent".to_owned(), format!("meeng-rust/{VERSION}")),
        ];
        let body = match body {
            Some(body) => {
                headers.push(("Content-Type".to_owned(), "application/json".to_owned()));
                Some(serde_json::to_vec(body).map_err(Error::Json)?)
            }
            None => None,
        };
        let key = options
            .idempotency_key
            .as_deref()
            .filter(|key| !key.is_empty())
            .map(str::to_owned);
        if let Some(key) = key {
            headers.push(("Idempotency-Key".to_owned(), key));
        }
        let may_repeat = matches!(
            method.to_ascii_uppercase().as_str(),
            "GET" | "HEAD" | "PUT" | "DELETE"
        ) || headers.iter().any(|(name, _)| name == "Idempotency-Key");
        let request = HttpRequest {
            method: method.to_owned(),
            url,
            headers,
            body,
            timeout: self.timeout,
        };
        let mut attempt = 0;
        let response = loop {
            match self.transport.send(request.clone()) {
                Ok(response) => {
                    let status = response.status;
                    let again =
                        status == 429 || (may_repeat && matches!(status, 500 | 502 | 503 | 504));
                    if again && attempt < self.max_retries {
                        self.pause(attempt, response.header("Retry-After"));
                        attempt += 1;
                        continue;
                    }
                    break response;
                }
                Err(failure) => {
                    // The connection failed or timed out: the call may or may not have arrived.
                    if !may_repeat || attempt >= self.max_retries {
                        return Err(Error::Transport(failure));
                    }
                    self.pause(attempt, None);
                    attempt += 1;
                }
            }
        };
        if response.status >= 300 {
            return Err(Error::Api(ApiError::from_response(&response)));
        }
        serde_json::from_slice(&response.body).map_err(Error::Json)
    }

    /// Sleeps before repetition number `attempt` + 1: the API's Retry-After, or
    /// the base delay doubled each time and spread over its upper half.
    fn pause(&self, attempt: u32, retry_after: Option<&str>) {
        let wait = match retry_after.and_then(|value| value.trim().parse::<f64>().ok()) {
            Some(seconds) if seconds > 0.0 => Duration::from_secs_f64(seconds),
            _ => {
                let delay = self
                    .retry_delay
                    .saturating_mul(1 << attempt.min(10))
                    .min(MAX_RETRY_DELAY);
                let half = delay / 2;
                let jitter = std::time::SystemTime::now()
                    .duration_since(std::time::UNIX_EPOCH)
                    .map_or(0, |elapsed| u64::from(elapsed.subsec_nanos()));
                half + Duration::from_nanos(jitter % (half.as_nanos().max(1) as u64 + 1))
            }
        };
        std::thread::sleep(wait.min(MAX_RETRY_DELAY));
    }
}

/// Says the base URL is https, or http to localhost, 127.0.0.0/8 or ::1.
fn secure_base_url(base_url: &str) -> bool {
    let Ok(uri) = base_url.parse::<ureq::http::Uri>() else {
        return false;
    };
    let host = uri
        .host()
        .unwrap_or("")
        .trim_matches(['[', ']'])
        .to_ascii_lowercase();
    let loopback = host == "localhost"
        || host
            .parse::<std::net::IpAddr>()
            .is_ok_and(|address| address.is_loopback());
    match uri.scheme_str().map(str::to_ascii_lowercase).as_deref() {
        Some("https") => true,
        Some("http") => loopback,
        _ => false,
    }
}

/// A path parameter or a query value, escaped: everything but letters,
/// digits and `-._~`.
pub(crate) fn escape(value: &str) -> String {
    const HEX: &[u8; 16] = b"0123456789ABCDEF";
    let mut out = String::with_capacity(value.len());
    for byte in value.bytes() {
        if byte.is_ascii_alphanumeric() || matches!(byte, b'-' | b'.' | b'_' | b'~') {
            out.push(byte as char);
        } else {
            out.push('%');
            out.push(HEX[usize::from(byte >> 4)] as char);
            out.push(HEX[usize::from(byte & 15)] as char);
        }
    }
    out
}
