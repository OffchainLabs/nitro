// Copyright 2021-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::js_core::{JsEnv, JsObject, JsValue};
use parking_lot::Mutex;
use std::io::Write;
use std::sync::Arc;

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

    object.insert_func("Object", |_env, _go, _args| Ok(JsObject::default().into()));
    object.insert_func("Array", |_env, _go, _args| {
        Ok(JsValue::Array(Default::default()))
    });
    object.insert("process", make_process_object());
    object.insert("fs", make_fs_object());
    object.insert_func("Uint8Array", |_env, _go, args| {
        if args.is_empty() {
            Ok(JsValue::Uint8Array(Default::default()))
        } else {
            let Some(JsValue::Number(size)) = args.first() else {
                panic!("Go trying to create new Uint8Array with bad args {args:?}")
            };
            if args.len() != 1 {
                eprintln!("Got incorrect number of arguments to new Uint8Array {args:?}");
            }
            Ok(JsValue::Uint8Array(Arc::new(Mutex::new(
                vec![0; *size as usize].into_boxed_slice(),
            ))))
        }
    });
    object.insert("crypto", make_crypto_object());
    object.insert_func("Date", |_env, _go, _args| Ok(make_date_object()));
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
    fs.insert_func("write", |env, _go, args| {
        // ignore any args after the 6th, and slice no more than than the number of args we have
        let args_len = std::cmp::min(6, args.len());
        let [
            JsValue::Number(fd),
            JsValue::Uint8Array(buf),
            JsValue::Number(offset),
            JsValue::Number(length),
            JsValue::Null,
            JsValue::Function(callback),
        ]  = &args[..args_len] else {
            panic!("Go trying to call fs.write with bad args {args:?}")
        };
        if args.len() != 6 {
            // Ignore any extra arguments but log a warning
            eprintln!("Got incorrect number of arguments to fs.write: {args:?}");
        }
        let buf = buf.lock();
        let mut offset = *offset as usize;
        let mut length = *length as usize;
        if offset > buf.len() {
            eprintln!(
                "Go trying to call fs.write with offset {offset} >= buf.len() {length}"
            );
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
        callback.call(env, JsValue::Undefined, Vec::new())?;
        Ok(JsValue::Undefined)
    });
    fs.into()
}

fn make_crypto_object() -> JsValue {
    let crypto = JsObject::default();
    crypto.insert_func("getRandomValues", |env, _go, args| {
        let Some(JsValue::Uint8Array(buf)) = args.first() else {
            panic!("Go trying to call crypto.getRandomValues with bad args {args:?}")
        };
        if args.len() != 1 {
            eprintln!("Got incorrect number of arguments to crypto.getRandomValues: {args:?}");
        }
        let mut buf = buf.lock();
        env.get_rng().fill_bytes(&mut buf);
        Ok(JsValue::Undefined)
    });
    crypto.into()
}

fn make_console_object() -> JsValue {
    let console = JsObject::default();
    console.insert_func("error", |_env, _go, args| {
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
    date.insert_func("getTimezoneOffset", |_env, _go, _args| {
        Ok(JsValue::Number(0.))
    });
    date.into()
}
