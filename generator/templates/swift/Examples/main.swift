// The examples of the Swift SDK, one program that runs the example it is asked for:
//
//   MEENG_API_KEY=mg_xxx swift run MeengExamples send-email
//
// send-email, send-batch, schedule-email, add-domain or verify-webhook. The key
// comes from MEENG_API_KEY and, to point the client at another app, the address
// from MEENG_BASE_URL.
import Foundation

let examples: [String: () async throws -> Void] = [
    "send-email": sendEmail,
    "send-batch": sendBatch,
    "schedule-email": scheduleEmail,
    "add-domain": addDomain,
    "verify-webhook": verifyWebhook,
]

guard CommandLine.arguments.count == 2, let example = examples[CommandLine.arguments[1]] else {
    FileHandle.standardError.write(Data("usage: MeengExamples \(examples.keys.sorted().joined(separator: " | "))\n".utf8))
    exit(2)
}

do {
    try await example()
} catch {
    FileHandle.standardError.write(Data("\(error)\n".utf8))
    exit(1)
}
