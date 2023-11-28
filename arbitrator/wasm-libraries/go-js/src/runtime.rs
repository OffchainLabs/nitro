// Copyright 2021-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use eyre::bail;

use crate::js_core::{JsEnv, JsObject, JsValue};
use std::io::Write;

macro_rules! match_args {
    ($args:expr, $name:expr, $count:expr, $($pat:pat),+) => {
        let [$($pat),+] = &$args[..$count] else {
            panic!("called {} with bad args: {:?}", $name, $args);
        };
        if $args.len() != $count {
            eprintln!("called {} with wrong number of args: {:?}", $name, $args);
        }
    };
}

pub fn make_go_object() -> JsValue {
    let object = JsObject::default();

    // Remove a warning if this is accessed before beign set
    object.insert("_pendingEvent", JsValue::Undefined);

    object.insert_func("_makeFuncWrapper", |_env, go, args| {
        if args.len() != 1 {
            eprintln!("Got incorrect arguments to _makeFuncWrapper: {args:?}");
        }
        let go = go.assume_object("go");
        let mut args = args.into_iter();
        let id = args.next().unwrap_or_default();
        let closure = move |env: &mut dyn JsEnv, this, args| {
            let event = JsObject::default();
            event.insert("id", id.clone());
            event.insert("this", this);
            event.insert("args", args);

            go.insert("_pendingEvent", JsValue::Object(event.clone()));
            env.resume()?;

            Ok(event.get("result").clone())
        };
        Ok(closure.into())
    });

    object.into()
}

pub fn make_globals_object() -> JsValue {
    let object = JsObject::default();

    object.insert_func(
        "Object",
        |_env, _this, _args| Ok(JsObject::default().into()),
    );
    object.insert_func("Array", |_env, _this, args| {
        if args.len() != 1 {
            return Ok(JsValue::from(args));
        }
        let JsValue::Number(len) = args[0] else {
            return Ok(JsValue::from(args));
        };
        if len.fract() != 0. {
            bail!("invalid array length");
        }
        Ok(JsValue::from(vec![JsValue::Number(0.); len as usize]))
    });
    object.insert("process", make_process_object());
    object.insert("fs", make_fs_object());
    object.insert_func("Uint8Array", |_env, _this, args| {
        if args.is_empty() {
            Ok(JsValue::Uint8Array(Default::default()))
        } else {
            match_args!(args, "new Uint8Array", 1, JsValue::Number(size));
            Ok(JsValue::new_uint8_array(vec![0; *size as usize]))
        }
    });
    object.insert("stylus", make_stylus_object());
    object.insert("crypto", make_crypto_object());
    object.insert_func("Date", |_env, _this, _args| Ok(make_date_object()));
    object.insert("console", make_console_object());
    // Triggers a code path in Go for a fake network impl
    object.insert("fetch", JsValue::Undefined);

    object.into()
}

fn make_process_object() -> JsValue {
    JsObject::default().into()
}

fn make_fs_object() -> JsValue {
    let constants = JsObject::default();
    for c in [
        "O_WRONLY", "O_RDWR", "O_CREAT", "O_TRUNC", "O_APPEND", "O_EXCL",
    ] {
        constants.insert(c, JsValue::Number(-1.));
    }

    let fs = JsObject::default();
    fs.insert("constants", constants);
    fs.insert_func("write", |env, _this, args| {
        match_args!(
            args,
            "fs.write",
            6,
            JsValue::Number(fd),
            JsValue::Uint8Array(buf),
            JsValue::Number(offset),
            JsValue::Number(length),
            JsValue::Null,
            JsValue::Function(callback)
        );
        let buf = buf.lock();
        let mut offset = *offset as usize;
        let mut length = *length as usize;
        if offset > buf.len() {
            eprintln!("Go trying to call fs.write with offset {offset} >= buf.len() {length}");
            offset = buf.len();
        }
        if offset + length > buf.len() {
            eprintln!(
                "Go trying to call fs.write with offset {offset} + length {length} >= buf.len() {}",
                buf.len(),
            );
            length = buf.len() - offset;
        }
        if *fd == 1. {
            let stdout = std::io::stdout();
            let mut stdout = stdout.lock();
            stdout.write_all(&buf[offset..(offset + length)]).unwrap();
        } else if *fd == 2. {
            let stderr = std::io::stderr();
            let mut stderr = stderr.lock();
            stderr.write_all(&buf[offset..(offset + length)]).unwrap();
        } else {
            eprintln!("Go trying to write to unknown FD {fd}");
        }
        // Don't borrow buf during the callback
        drop(buf);

        let args = vec![
            JsValue::Null,                  // no error
            JsValue::Number(length as f64), // amount written
        ];
        callback.call(env, JsValue::Undefined, args)?;
        Ok(JsValue::Undefined)
    });
    fs.into()
}

fn make_crypto_object() -> JsValue {
    let crypto = JsObject::default();
    crypto.insert_func("getRandomValues", |env, _this, args| {
        match_args!(args, "crypto.getRandomValues", 1, JsValue::Uint8Array(buf));
        let mut buf = buf.lock();
        env.get_rng().fill_bytes(&mut buf);
        Ok(JsValue::Undefined)
    });
    crypto.into()
}

fn make_console_object() -> JsValue {
    let console = JsObject::default();
    console.insert_func("error", |_env, _this, args| {
        eprintln!("Go console error:");
        for arg in args {
            eprintln!("{arg:?}");
        }
        eprintln!();
        Ok(JsValue::Undefined)
    });
    console.into()
}

fn make_date_object() -> JsValue {
    let date = JsObject::default();
    date.insert_func("getTimezoneOffset", |_env, _this, _args| {
        Ok(JsValue::Number(0.))
    });
    date.into()
}

fn make_stylus_object() -> JsValue {
    JsObject::default().into()
}
