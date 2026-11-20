using System;
using System.Collections.Generic;
using System.Text.Json;
using System.Text.Json.Serialization;

namespace Meeng;

/// <summary>
/// Reads and writes the lists of strings the API also takes, and may answer, as
/// one string: the recipients of an email. It writes a list; a lone string is
/// read as a list of one.
/// </summary>
internal sealed class StringListConverter : JsonConverter<IReadOnlyList<string>>
{
    public override IReadOnlyList<string>? Read(ref Utf8JsonReader reader, Type typeToConvert, JsonSerializerOptions options)
    {
        if (reader.TokenType == JsonTokenType.String)
        {
            return [reader.GetString()!];
        }
        if (reader.TokenType != JsonTokenType.StartArray)
        {
            throw new JsonException("meeng: expected a string or a list of strings");
        }
        var list = new List<string>();
        while (reader.Read() && reader.TokenType != JsonTokenType.EndArray)
        {
            if (reader.TokenType != JsonTokenType.String)
            {
                throw new JsonException("meeng: expected a string in the list");
            }
            list.Add(reader.GetString()!);
        }
        return list;
    }

    public override void Write(Utf8JsonWriter writer, IReadOnlyList<string> value, JsonSerializerOptions options)
    {
        writer.WriteStartArray();
        foreach (var each in value)
        {
            writer.WriteStringValue(each);
        }
        writer.WriteEndArray();
    }
}
