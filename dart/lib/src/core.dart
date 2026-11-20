/// What a service calls the API through: [Meeng], or a fake in a test.
abstract interface class Requester {
  /// Calls the API and answers its JSON, decoded: a map, a list, or a scalar.
  ///
  /// A null or empty value in [query] is left out, and a [List] in it is sent
  /// once for each value. [idempotencyKey] makes a send safe to retry.
  Future<Object?> request(
    String method,
    String path, {
    Map<String, Object?> query,
    Object? body,
    String? idempotencyKey,
  });
}
