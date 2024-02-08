fn main() {
    println!("cargo:rustc-link-lib=static=hashtree");
    println!("cargo:rustc-link-search=native=/usr/lib");
}
