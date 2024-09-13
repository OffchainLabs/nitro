use arbutil::hostios::HOSTIOS;
use std::{env, fmt::Write, fs, path::Path};

pub fn gen_forwarder(out_path: &Path) {
    let mut wat = String::new();
    macro_rules! wln {
        ($($text:tt)*) => {
            writeln!(wat, $($text)*).unwrap();
        };
    }
    let s = "    ";

    wln!("(module");

    macro_rules! group {
        ($list:expr, $kind:expr) => {
            (!$list.is_empty())
                .then(|| {
                    format!(
                        " ({} {})",
                        $kind,
                        $list
                            .iter()
                            .map(|x| x.to_string())
                            .collect::<Vec<_>>()
                            .join(" ")
                    )
                })
                .unwrap_or_default()
        };
    }

    wln!("{s};; symbols to re-export");
    for (name, ins, outs) in HOSTIOS {
        let params = group!(ins, "param");
        let result = group!(outs, "result");
        wln!(r#"{s}(import "user_host" "{name}" (func $_{name}{params}{result}))"#);
    }
    wln!();

    wln!("{s};; reserved offsets for future user_host imports");
    for i in HOSTIOS.len()..512 {
        wln!("{s}(func $reserved_{i} unreachable)");
    }
    wln!();

    wln!(
        "{s};; allows user_host to request a trap\n\
        {s}(global $trap (mut i32) (i32.const 0))\n\
        {s}(func $check\n\
        {s}{s}global.get $trap                    ;; see if set\n\
        {s}{s}(global.set $trap (i32.const 0))    ;; reset the flag\n\
        {s}{s}(if (then (unreachable)))\n\
        {s})\n\
        {s}(func (export \"forward__set_trap\")\n\
        {s}{s}(global.set $trap (i32.const 1))\n\
        {s})\n"
    );

    wln!("{s};; user linkage");
    for (name, ins, outs) in HOSTIOS {
        let params = group!(ins, "param");
        let result = group!(outs, "result");
        wln!("{s}(func (export \"vm_hooks__{name}\"){params}{result}");

        for i in 0..ins.len() {
            wln!("{s}{s}local.get {i}");
        }

        wln!(
            "{s}{s}call $_{name}\n\
             {s}{s}call $check\n\
             {s})"
        );
    }

    wln!(")");
    eprintln!("{}", &wat);

    let wasm = wasmer::wat2wasm(wat.as_bytes()).unwrap();

    fs::write(out_path, wasm.as_ref()).unwrap();
}

fn main() {
    let out_dir = env::var("OUT_DIR").unwrap();
    let forwarder_path = Path::new(&out_dir).join("forwarder.wasm");
    gen_forwarder(&forwarder_path);
}
