fn main() {
    println!("cargo:rerun-if-env-changed=ONEDAY_UPDATER_ENDPOINT");
    println!("cargo:rerun-if-env-changed=ONEDAY_UPDATER_PUBKEY");
    tauri_build::build()
}
