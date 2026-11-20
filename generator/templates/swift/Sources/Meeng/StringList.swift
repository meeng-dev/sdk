/// A list of strings the API takes, and may answer, as one string or as a list:
/// the recipients of an email. It is sent as a list; a lone string is read as a
/// list of one.
public struct StringList: Codable, Sendable, Hashable, RandomAccessCollection, ExpressibleByArrayLiteral,
    ExpressibleByStringLiteral
{
    /// The strings.
    public var elements: [String]

    public init(_ elements: [String]) {
        self.elements = elements
    }

    public init(arrayLiteral elements: String...) {
        self.elements = elements
    }

    public init(stringLiteral value: String) {
        self.elements = [value]
    }

    public init(from decoder: any Decoder) throws {
        let container = try decoder.singleValueContainer()
        if let one = try? container.decode(String.self) {
            elements = [one]
        } else {
            elements = try container.decode([String].self)
        }
    }

    public func encode(to encoder: any Encoder) throws {
        var container = encoder.singleValueContainer()
        try container.encode(elements)
    }

    public var startIndex: Int { elements.startIndex }
    public var endIndex: Int { elements.endIndex }

    public subscript(position: Int) -> String { elements[position] }
}
