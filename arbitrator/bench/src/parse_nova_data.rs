use std::io::{self, BufRead};

#[derive(Debug, Default)]
pub struct GlobalState {
    pub block_hash: Vec<u8>,
    pub send_root: Vec<u8>,
    pub batch: u64,
    pub pos_in_batch: u64,
}

fn parse_to_byte_slice(input: &str) -> Result<Vec<u8>, std::num::ParseIntError> {
    input
        .trim_matches(|p| p == '[' || p == ']') // Remove the brackets
        .split_whitespace() // Split by whitespace
        .map(|s| s.parse::<u8>()) // Parse each string as a byte
        .collect() // Collect into a Result<Vec<u8>, ParseIntError>
}

#[derive(Debug, Default)]
pub struct FileData {
    pub id: u64,
    pub pos: u64,
    pub msg: Vec<u8>,
    pub has_delayed_msg: bool,
    pub delayed_msg_nr: u64,
    pub delayed_msg: Vec<u8>,
    pub delayed_messages_read: u64,
    pub start_state: GlobalState,
    pub end_state: GlobalState,
    pub header: Header,
}

#[derive(Debug, Default)]
pub struct Header {
    pub kind: u64,
    pub poster: Vec<u8>,
    pub block_number: u64,
    pub timestamp: u64,
    pub l1_basefee: u64,
}

impl FileData {
    pub fn from_reader<R: BufRead>(mut reader: R) -> io::Result<Self> {
        let mut line = String::new();
        let mut start_state = GlobalState::default();
        let mut end_state = GlobalState::default();
        let mut header = Header::default();
        let mut data = FileData::default();
        while reader.read_line(&mut line)? > 0 {
            if line.starts_with("pos:") {
                data.pos = line.split(":").nth(1).unwrap().trim().parse().unwrap();
            }
            if line.starts_with("start-block-hash:") {
                start_state.block_hash =
                    hex::decode(line.split(":").nth(1).unwrap().trim()).unwrap();
            }
            if line.starts_with("start-send-root:") {
                start_state.send_root =
                    hex::decode(line.split(":").nth(1).unwrap().trim()).unwrap();
            }
            if line.starts_with("start-batch:") {
                start_state.batch = line.split(":").nth(1).unwrap().trim().parse().unwrap();
            }
            if line.starts_with("start-pos-in-batch:") {
                start_state.pos_in_batch = line.split(":").nth(1).unwrap().trim().parse().unwrap();
            }
            if line.starts_with("end-block-hash:") {
                end_state.block_hash = hex::decode(line.split(":").nth(1).unwrap().trim()).unwrap();
            }
            if line.starts_with("end-send-root:") {
                end_state.send_root = hex::decode(line.split(":").nth(1).unwrap().trim()).unwrap();
            }
            if line.starts_with("end-batch:") {
                end_state.batch = line.split(":").nth(1).unwrap().trim().parse().unwrap();
            }
            if line.starts_with("end-pos-in-batch:") {
                end_state.pos_in_batch = line.split(":").nth(1).unwrap().trim().parse().unwrap();
            }
            if line.starts_with("has-delayed-msg:") {
                data.has_delayed_msg = match line.split(":").nth(1).unwrap().trim() {
                    "true" => true,
                    "false" => false,
                    _ => false,
                }
            }
            if line.starts_with("delayed-msg-nr:") {
                data.delayed_msg_nr = line.split(":").nth(1).unwrap().trim().parse().unwrap();
            }
            if line.starts_with("l2-msg:") {
                data.msg = parse_to_byte_slice(line.split(":").nth(1).unwrap().trim()).unwrap();
            }
            if line.starts_with("kind:") {
                header.kind = line.split(":").nth(1).unwrap().trim().parse().unwrap();
            }
            if line.starts_with("poster:") {
                header.poster = hex::decode(line.split(":").nth(1).unwrap().trim()).unwrap();
            }
            if line.starts_with("block-number:") {
                header.block_number = line.split(":").nth(1).unwrap().trim().parse().unwrap();
            }
            if line.starts_with("timestamp:") {
                header.timestamp = line.split(":").nth(1).unwrap().trim().parse().unwrap();
            }
            if line.starts_with("l1-basefee:") {
                header.l1_basefee = line.split(":").nth(1).unwrap().trim().parse().unwrap();
            }
            if line.starts_with("msg-delayed-messages-read:") {
                data.delayed_messages_read =
                    line.split(":").nth(1).unwrap().trim().parse().unwrap();
            }
            line.clear();
        }
        data.start_state = start_state;
        data.end_state = end_state;
        data.header = header;
        Ok(data)
    }
}
