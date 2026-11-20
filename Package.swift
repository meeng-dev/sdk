// swift-tools-version:6.0
// SwiftPM reads a package at the root of its repository, so this file exposes
// the Swift SDK that sdk/generator writes under swift/ to
// .package(url: "https://github.com/meeng-dev/sdk", from: "X.Y.Z"). It is
// written by hand: keep it equal to swift/Package.swift but for the paths.
import PackageDescription

let package = Package(
    name: "Meeng",
    platforms: [.macOS(.v12), .iOS(.v15), .tvOS(.v15), .watchOS(.v8), .visionOS(.v1)],
    products: [
        .library(name: "Meeng", targets: ["Meeng"])
    ],
    targets: [
        .target(name: "Meeng", path: "swift/Sources/Meeng"),
        .testTarget(name: "MeengTests", dependencies: ["Meeng"], path: "swift/Tests/MeengTests"),
    ]
)
