use std::io::{self, BufRead};

#[derive(Debug)]
pub struct BatchInfo {
    pub number: u64,
    pub data: Vec<u8>,
}

#[derive(Debug, Default)]
pub struct GlobalState {
    pub block_hash: Vec<u8>,
    pub send_root: Vec<u8>,
    pub batch: u64,
    pub pos_in_batch: u64,
}

#[derive(Debug)]
pub struct FileData {
    pub id: u64,
    pub has_delayed_msg: bool,
    pub delayed_msg_nr: u64,
    pub delayed_msg: Vec<u8>,
}

impl FileData {
    pub fn from_reader<R: BufRead>(mut reader: R) -> io::Result<Self> {
        let mut line = String::new();
        let mut start_state = GlobalState::default();
        while reader.read_line(&mut line)? > 0 {
            if line.starts_with("start-block-hash:") {
                dbg!(line.split(":").nth(1).unwrap().trim());
                start_state.block_hash =
                    hex::decode(line.split(":").nth(1).unwrap().trim()).unwrap();
            }
            // if line.starts_with("Id:") {
            // } else if line.starts_with("HasDelayedMsg:") {
            //     has_delayed_msg = line.split(":").nth(1).unwrap().trim().parse().unwrap();
            // } else if line.starts_with("DelayedMsgNr:") {
            //     delayed_msg_nr = line.split(":").nth(1).unwrap().trim().parse().unwrap();
            // } else if line.starts_with("Preimages:") {
            //     items.push(Item::from_reader(&mut reader, &mut line)?);
            // } else if line.starts_with("BatchInfo:") {
            //     let parts: Vec<_> = line.split(",").collect();
            //     batch_info.number = parts[0].split(":").nth(2).unwrap().trim().parse().unwrap();
            //     batch_info.data = hex::decode(parts[1].split(":").nth(1).unwrap().trim()).unwrap();
            // } else if line.starts_with("DelayedMsg:") {
            //     delayed_msg = hex::decode(line.split(":").nth(1).unwrap().trim()).unwrap();
            // } else if line.starts_with("StartState:") {
            //     let parts: Vec<_> = line.split(",").collect();

            //     // Parsing block_hash
            //     let block_hash_str = parts[0].split("BlockHash:").nth(1).unwrap().trim();
            //     start_state.block_hash =
            //         hex::decode(block_hash_str.strip_prefix("0x").unwrap()).unwrap();

            //     // Parsing send_root
            //     let send_root_str = parts[1].split(":").nth(1).unwrap().trim();
            //     start_state.send_root =
            //         hex::decode(send_root_str.strip_prefix("0x").unwrap()).unwrap();

            //     // Parsing batch
            //     start_state.batch = parts[2]
            //         .split(":")
            //         .nth(1)
            //         .unwrap()
            //         .trim()
            //         .parse::<u64>()
            //         .unwrap();

            //     // Parsing pos_in_batch
            //     start_state.pos_in_batch = parts[3]
            //         .split(":")
            //         .nth(1)
            //         .unwrap()
            //         .trim()
            //         .parse::<u64>()
            //         .unwrap();
            // }
            line.clear();
        }

        dbg!(start_state);
        Ok(FileData {
            id: 0,
            has_delayed_msg: false,
            delayed_msg_nr: 0,
            delayed_msg: vec![],
        })
    }
}
