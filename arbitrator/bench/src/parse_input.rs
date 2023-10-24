use std::io::{self, BufRead};

#[derive(Debug, Clone)]
pub struct Preimage {
    pub type_: u32,
    pub hash: Vec<u8>,
    pub data: Vec<u8>,
}

#[derive(Debug, Clone)]
pub struct Item {
    pub preimages: Vec<Preimage>,
}

#[derive(Debug)]
pub struct BatchInfo {
    pub number: u64,
    pub data: Vec<u8>,
}

#[derive(Debug)]
pub struct StartState {
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
    pub items: Vec<Item>,
    pub batch_info: BatchInfo,
    pub delayed_msg: Vec<u8>,
    pub start_state: StartState,
}

impl FileData {
    pub fn from_reader<R: BufRead>(mut reader: R) -> io::Result<Self> {
        let mut items = Vec::new();
        let mut batch_info = BatchInfo {
            number: 0,
            data: Vec::new(),
        };
        let mut id = 0;
        let mut has_delayed_msg = false;
        let mut delayed_msg_nr = 0;
        let mut delayed_msg = Vec::new();
        let mut start_state = StartState {
            block_hash: Vec::new(),
            send_root: Vec::new(),
            batch: 0,
            pos_in_batch: 0,
        };

        let mut line = String::new();
        while reader.read_line(&mut line)? > 0 {
            if line.starts_with("Id:") {
                id = line.split(":").nth(1).unwrap().trim().parse().unwrap();
            } else if line.starts_with("HasDelayedMsg:") {
                has_delayed_msg = line.split(":").nth(1).unwrap().trim().parse().unwrap();
            } else if line.starts_with("DelayedMsgNr:") {
                delayed_msg_nr = line.split(":").nth(1).unwrap().trim().parse().unwrap();
            } else if line.starts_with("Preimages:") {
                items.push(Item::from_reader(&mut reader, &mut line)?);
            } else if line.starts_with("BatchInfo:") {
                let parts: Vec<_> = line.split(",").collect();
                batch_info.number = parts[0].split(":").nth(2).unwrap().trim().parse().unwrap();
                batch_info.data = hex::decode(parts[1].split(":").nth(1).unwrap().trim()).unwrap();
            } else if line.starts_with("DelayedMsg:") {
                delayed_msg = hex::decode(line.split(":").nth(1).unwrap().trim()).unwrap();
            } else if line.starts_with("StartState:") {
                let parts: Vec<_> = line.split(",").collect();

                // Parsing block_hash
                let block_hash_str = parts[0].split("BlockHash:").nth(1).unwrap().trim();
                start_state.block_hash =
                    hex::decode(block_hash_str.strip_prefix("0x").unwrap()).unwrap();

                // Parsing send_root
                let send_root_str = parts[1].split(":").nth(1).unwrap().trim();
                start_state.send_root =
                    hex::decode(send_root_str.strip_prefix("0x").unwrap()).unwrap();

                // Parsing batch
                start_state.batch = parts[2]
                    .split(":")
                    .nth(1)
                    .unwrap()
                    .trim()
                    .parse::<u64>()
                    .unwrap();

                // Parsing pos_in_batch
                start_state.pos_in_batch = parts[3]
                    .split(":")
                    .nth(1)
                    .unwrap()
                    .trim()
                    .parse::<u64>()
                    .unwrap();
            }
            line.clear();
        }

        Ok(FileData {
            id,
            has_delayed_msg,
            delayed_msg_nr,
            items,
            batch_info,
            delayed_msg,
            start_state,
        })
    }
}

impl Item {
    pub fn from_reader<R: BufRead>(reader: &mut R, line: &mut String) -> io::Result<Self> {
        let mut preimages = Vec::new();

        loop {
            if line.is_empty()
                || line.starts_with("BatchInfo:")
                || line.starts_with("DelayedMsg:")
                || line.starts_with("StartState:")
            {
                break;
            }
            if line.starts_with("Preimages:") {
                line.clear();
                while reader.read_line(line)? > 0 && line.starts_with("\t") {
                    let parts: Vec<_> = line.trim().split(",").collect();
                    let type_ = parts[0].split(":").nth(1).unwrap().trim().parse().unwrap();
                    let hash = hex::decode(
                        parts[1]
                            .split(":")
                            .nth(1)
                            .unwrap()
                            .trim()
                            .strip_prefix("0x")
                            .unwrap(),
                    )
                    .unwrap();
                    let data = hex::decode(parts[2].split(":").nth(1).unwrap().trim()).unwrap();
                    preimages.push(Preimage { type_, hash, data });
                    line.clear();
                }
                continue; // To skip line.clear() at the end of the loop for this case
            }

            line.clear();
            if reader.read_line(line)? == 0 {
                // If EOF is reached
                break;
            }
        }

        Ok(Item { preimages })
    }
}
