// swift-tools-version:6.0
// meeng's SDK for Swift, written by sdk/generator; do not edit. The
// repository's root Package.swift builds these same sources, so SwiftPM can
// depend on https://github.com/meeng-dev/sdk.
import PackageDescription

let package = Package(
    name: "Meeng",
    platforms: [.macOS(.v12), .iOS(.v15), .tvOS(.v15), .watchOS(.v8), .visionOS(.v1)],
    products: [
        .library(name: "Meeng", targets: ["Meeng"])
    ],
    targets: [
        .target(name: "Meeng"),
        .testTarget(name: "MeengTests", dependencies: ["Meeng"]),
    ]
)
