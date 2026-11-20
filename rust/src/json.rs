//! Reading the lists of strings the API also writes as one string.

use serde::{Deserialize, Deserializer};

#[derive(Deserialize)]
#[serde(untagged)]
enum OneOrMany {
    One(String),
    Many(Vec<String>),
}

impl From<OneOrMany> for Vec<String> {
    fn from(value: OneOrMany) -> Self {
        match value {
            OneOrMany::One(one) => vec![one],
            OneOrMany::Many(many) => many,
        }
    }
}

/// A list of strings, or one string read as a list of one.
pub(crate) fn string_list<'de, D: Deserializer<'de>>(
    deserializer: D,
) -> Result<Vec<String>, D::Error> {
    OneOrMany::deserialize(deserializer).map(Vec::from)
}

/// [`string_list`], or null.
pub(crate) fn optional_string_list<'de, D: Deserializer<'de>>(
    deserializer: D,
) -> Result<Option<Vec<String>>, D::Error> {
    Option::<OneOrMany>::deserialize(deserializer).map(|value| value.map(Vec::from))
}
