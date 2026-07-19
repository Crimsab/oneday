fn main() {
    println!("cargo:rerun-if-env-changed=ONEDAY_UPDATER_ENDPOINT");
    println!("cargo:rerun-if-env-changed=ONEDAY_UPDATER_PUBKEY");
    println!("cargo:rerun-if-env-changed=TARGET");
    println!("cargo:rerun-if-changed=Cargo.toml");
    println!(
        "cargo:rustc-env=ONEDAY_TARGET_TRIPLE={}",
        std::env::var("TARGET").expect("Cargo target")
    );
    tauri_build::build()
}
