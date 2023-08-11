use fnv::FnvHashMap as HashMap;

pub const ZERO_ID: u32 = 1;
pub const NULL_ID: u32 = 2;
pub const GLOBAL_ID: u32 = 5;
pub const GO_ID: u32 = 6;

pub const OBJECT_ID: u32 = 100;
pub const ARRAY_ID: u32 = 101;
pub const PROCESS_ID: u32 = 102;
pub const FS_ID: u32 = 103;
pub const UINT8_ARRAY_ID: u32 = 104;
pub const CRYPTO_ID: u32 = 105;
pub const DATE_ID: u32 = 106;

pub const FS_CONSTANTS_ID: u32 = 200;

pub const DYNAMIC_OBJECT_ID_BASE: u32 = 10000;

#[derive(Clone, Copy, Debug, PartialEq)]
pub enum InterpValue {
    Undefined,
    Number(f64),
    Ref(u32),
}

impl InterpValue {
    pub fn assume_num_or_object(self) -> GoValue {
        match self {
            InterpValue::Undefined => GoValue::Undefined,
            InterpValue::Number(x) => GoValue::Number(x),
            InterpValue::Ref(x) => GoValue::Object(x),
        }
    }
}

#[derive(Clone, Copy, Debug)]
#[allow(dead_code)]
pub enum GoValue {
    Undefined,
    Number(f64),
    Null,
    Object(u32),
    String(u32),
    Symbol(u32),
    Function(u32),
}

impl GoValue {
    pub fn encode(self) -> u64 {
        let (ty, id): (u32, u32) = match self {
            GoValue::Undefined => return 0,
            GoValue::Number(mut f) => {
                // Canonicalize NaNs so they don't collide with other value types
                if f.is_nan() {
                    f = f64::NAN;
                }
                if f == 0. {
                    // Zeroes are encoded differently for some reason
                    (0, ZERO_ID)
                } else {
                    return f.to_bits();
                }
            }
            GoValue::Null => (0, NULL_ID),
            GoValue::Object(x) => (1, x),
            GoValue::String(x) => (2, x),
            GoValue::Symbol(x) => (3, x),
            GoValue::Function(x) => (4, x),
        };
        // Must not be all zeroes, otherwise it'd collide with a real NaN
        assert!(ty != 0 || id != 0, "GoValue must not be empty");
        f64::NAN.to_bits() | (u64::from(ty) << 32) | u64::from(id)
    }
}

#[derive(Clone, Debug)]
pub struct PendingEvent {
    pub id: InterpValue,
    pub this: InterpValue,
    pub args: Vec<GoValue>,
}

#[derive(Debug, Clone)]
pub enum DynamicObject {
    Uint8Array(Vec<u8>),
    FunctionWrapper(InterpValue, InterpValue),
    PendingEvent(PendingEvent),
    ValueArray(Vec<GoValue>),
    Date,
}

#[derive(Default, Debug)]
pub struct DynamicObjectPool {
    objects: HashMap<u32, DynamicObject>,
    free_ids: Vec<u32>,
}

static mut DYNAMIC_OBJECT_POOL: Option<DynamicObjectPool> = None;

impl DynamicObjectPool {
    pub unsafe fn singleton<'a>() -> &'a mut Self {
        DYNAMIC_OBJECT_POOL.get_or_insert_with(Default::default)
    }

    pub fn insert(&mut self, object: DynamicObject) -> u32 {
        let id = self
            .free_ids
            .pop()
            .unwrap_or_else(|| DYNAMIC_OBJECT_ID_BASE + self.objects.len() as u32);
        self.objects.insert(id, object);
        id
    }

    pub fn get(&self, id: u32) -> Option<&DynamicObject> {
        self.objects.get(&id)
    }

    pub fn get_mut(&mut self, id: u32) -> Option<&mut DynamicObject> {
        self.objects.get_mut(&id)
    }

    pub fn remove(&mut self, id: u32) -> Option<DynamicObject> {
        let res = self.objects.remove(&id);
        if res.is_some() {
            self.free_ids.push(id);
        }
        res
    }
}

pub static mut PENDING_EVENT: Option<PendingEvent> = None;

pub unsafe fn get_field(source: u32, field: &[u8]) -> GoValue {
    if source == GLOBAL_ID {
        if field == b"Object" {
            return GoValue::Function(OBJECT_ID);
        } else if field == b"Array" {
            return GoValue::Function(ARRAY_ID);
        } else if field == b"process" {
            return GoValue::Object(PROCESS_ID);
        } else if field == b"fs" {
            return GoValue::Object(FS_ID);
        } else if field == b"Uint8Array" {
            return GoValue::Function(UINT8_ARRAY_ID);
        } else if field == b"crypto" {
            return GoValue::Object(CRYPTO_ID);
        } else if field == b"Date" {
            return GoValue::Object(DATE_ID);
        } else if field == b"fetch" {
            // Triggers a code path in Go for a fake network implementation
            return GoValue::Undefined;
        }
    } else if source == FS_ID {
        if field == b"constants" {
            return GoValue::Object(FS_CONSTANTS_ID);
        }
    } else if source == FS_CONSTANTS_ID {
        if matches!(
            field,
            b"O_WRONLY" | b"O_RDWR" | b"O_CREAT" | b"O_TRUNC" | b"O_APPEND" | b"O_EXCL"
        ) {
            return GoValue::Number(-1.);
        }
    } else if source == GO_ID {
        if field == b"_pendingEvent" {
            if let Some(event) = &PENDING_EVENT {
                let id = DynamicObjectPool::singleton()
                    .insert(DynamicObject::PendingEvent(event.clone()));
                return GoValue::Object(id);
            } else {
                return GoValue::Null;
            }
        }
    } else if source == PROCESS_ID {
        if field == b"pid" {
            return GoValue::Number(1.);
        }
    }

    if let Some(source) = DynamicObjectPool::singleton().get(source).cloned() {
        if let DynamicObject::PendingEvent(event) = &source {
            if field == b"id" {
                return event.id.assume_num_or_object();
            } else if field == b"this" {
                return event.this.assume_num_or_object();
            } else if field == b"args" {
                let id = DynamicObjectPool::singleton()
                    .insert(DynamicObject::ValueArray(event.args.clone()));
                return GoValue::Object(id);
            }
        }

        eprintln!(
            "Go attempting to access unimplemented unknown JS value {:?} field {}",
            source,
            String::from_utf8_lossy(field),
        );
        GoValue::Undefined
    } else {
        eprintln!(
            "Go attempting to access unimplemented unknown JS value {} field {}",
            source,
            String::from_utf8_lossy(field),
        );
        GoValue::Undefined
    }
}
