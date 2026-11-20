import 'dart:io' show Platform;

/// The environment variable [name], or null when it is not set or is empty.
String? environment(String name) {
  final value = Platform.environment[name];
  return value == null || value.isEmpty ? null : value;
}
