import 'dart:convert';

/// A request the API refused.
///
/// [code] is the API's, such as `NOT_UNIQUE`, `FORBIDDEN` or `NOT_FOUND`; it is
/// `INTERNAL_SERVER_ERROR` when the answer carried none. [retryAfter] is how
/// many seconds to wait on a 429.
class MeengException implements Exception {
  const MeengException(this.status, this.code, this.message, {this.retryAfter});

  /// The error a refused response carries: its status, headers and body, as
  /// `{"error": {"code", "message"}}`, `{"error": "text"}` or the error itself.
  factory MeengException.fromResponse(
    int status,
    Map<String, String> headers,
    List<int> body,
  ) {
    String? code;
    String? message;
    try {
      final decoded = jsonDecode(utf8.decode(body));
      if (decoded is Map<String, Object?>) {
        final found = decoded.containsKey('error') ? decoded['error'] : decoded;
        if (found is String) {
          message = found;
        } else if (found is Map<String, Object?>) {
          code = found['code'] is String ? found['code'] as String : null;
          message = found['message'] is String
              ? found['message'] as String
              : null;
          // A refused validation answers
          // {"success": false, "error": {"issues", "name"}}, with no code of
          // its own.
          if (code == null || code.isEmpty) {
            code = found['name'] is String ? found['name'] as String : null;
          }
          if (message == null || message.isEmpty) {
            final issues = found['issues'];
            if (issues is List<Object?> && issues.isNotEmpty) {
              final first = issues.first;
              if (first is Map<String, Object?> && first['message'] is String) {
                final path = first['path'];
                final where = path is List<Object?> && path.isNotEmpty
                    ? '${path.join('.')}: '
                    : '';
                message = '$where${first['message']}';
              }
            }
          }
        }
      }
    } on FormatException {
      // A refusal without a JSON body is still a refusal.
    }
    return MeengException(
      status,
      code == null || code.isEmpty ? 'INTERNAL_SERVER_ERROR' : code,
      message == null || message.isEmpty ? _reason(status) : message,
      retryAfter: int.tryParse(_header(headers, 'retry-after')),
    );
  }

  /// The HTTP status.
  final int status;

  /// The API's error code.
  final String code;

  /// What went wrong, in words.
  final String message;

  /// Seconds to wait before trying again, on a 429.
  final int? retryAfter;

  @override
  String toString() => 'meeng: $status $code: $message';
}

String _reason(int status) => switch (status) {
  400 => 'Bad Request',
  401 => 'Unauthorized',
  403 => 'Forbidden',
  404 => 'Not Found',
  409 => 'Conflict',
  422 => 'Unprocessable Entity',
  429 => 'Too Many Requests',
  500 => 'Internal Server Error',
  502 => 'Bad Gateway',
  503 => 'Service Unavailable',
  _ => 'Error',
};

String _header(Map<String, String> headers, String name) {
  for (final MapEntry(:key, :value) in headers.entries) {
    if (key.toLowerCase() == name) {
      return value.trim();
    }
  }
  return '';
}
