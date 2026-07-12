use std::{env, fs, path::PathBuf};
use typify::{TypeSpace, TypeSpaceSettings};

fn main() -> Result<(), Box<dyn std::error::Error>> {
    let manifest_dir = PathBuf::from(env::var("CARGO_MANIFEST_DIR")?);
    let schema_path = manifest_dir.join("../contracts/gateway-v1.schema.json");
    println!("cargo:rerun-if-changed={}", schema_path.display());

    let schema: schemars::schema::RootSchema =
        serde_json::from_str(&fs::read_to_string(&schema_path)?)?;
    let settings = TypeSpaceSettings::default();
    let mut type_space = TypeSpace::new(&settings);
    type_space.add_root_schema(schema)?;

    let syntax = syn::parse2::<syn::File>(type_space.to_stream())?;
    let output = PathBuf::from(env::var("OUT_DIR")?).join("gateway_protocol.rs");
    fs::write(output, prettyplease::unparse(&syntax))?;
    Ok(())
}
