// Copyright 2022-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use activate::{util, wasm, Trial};
use arbutil::{color, format};
use eyre::{bail, Result};
use rand::{seq::SliceRandom, Rng};
use std::{fs::File, io::Write, ptr};

pub fn attack() -> Result<()> {
    util::set_cpu_affinity(&[2]);
    let pop_size = 100;

    let mut rng = rand::thread_rng();

    let mut pop = vec![];
    for _ in 0..pop_size {
        pop.push(Org::new_random());
    }

    let mut last_best = i64::MIN;
    let mut last_gain = 0;

    for gen in 0.. {
        pop.iter_mut().for_each(|org| {
            org.fitness = org.eval().ok();
        });

        // drop worst half
        pop.retain(|x| x.fitness.is_some());
        pop.sort_by_key(|x| -x.fitness.unwrap());
        pop.truncate(1 * pop.len() / 2);

        while pop.len() < pop_size {
            if rng.gen_ratio(1, 8) {
                pop.push(Org::new_random());
                continue;
            }

            let a = pop.choose(&mut rng).unwrap();
            let b = pop.choose(&mut rng).unwrap();
            let mut child = Org::cross(a, b);
            child.mutate();
            pop.push(child);
        }

        let best = pop[0].fitness.unwrap();
        let best_trial = Trial::new(&pop[0].wasm()?)?;
        let best_name = best_trial.accuracy().0;

        println!(
            "gen {gen} best: {} {best_name}, {}",
            color::when(best > 0, best, color::RED),
            format::bytes(pop[0].wasm()?.len())
        );

        if best > last_best {
            last_best = best;
            last_gain = gen;
        }
        if gen - last_gain > 100 {
            println!("resetting...");
            last_gain = gen;
            last_best = i64::MIN;
            for i in 0..pop_size {
                pop[i] = Org::new_random();
            }
        }

        if best > 0 {
            let wat = wasm::wat(&pop[0].wasm()?).unwrap_or("???".into());
            let mut file = File::create("attack.wat")?;
            writeln!(file, "{wat}")?;
        }
    }

    Ok(())
}

struct Org {
    data: Vec<u8>,
    fitness: Option<i64>,
}

impl Org {
    const SIZE_LIMIT: usize = 128 * 1024;

    fn new_random() -> Self {
        let mut rng = rand::thread_rng();
        let len = rng.gen_range(0..Self::SIZE_LIMIT);
        Org {
            data: util::random_vec(len),
            fitness: None,
        }
    }

    fn wasm(&self) -> Result<Vec<u8>> {
        let wasm = wasm::random(&self.data)?;
        if wasm.len() > Self::SIZE_LIMIT {
            bail!("too large");
        }
        Ok(wasm)
    }

    fn eval(&self) -> Result<i64> {
        if let Some(fitness) = self.fitness {
            return Ok(fitness);
        }

        let mut fitness = i64::MIN;

        for _ in 0..4 {
            let trial = Trial::new(&self.wasm()?)?;
            fitness = trial.accuracy().1;
            if fitness < 0 {
                return Ok(fitness);
            }
        }
        Ok(fitness)
    }

    fn mutate(&mut self) {
        let mut rng = rand::thread_rng();
        let points = rng.gen::<usize>() % (1 + self.data.len() / 16);
        for _ in 0..points {
            let byte = self.data.choose_mut(&mut rng);
            byte.map(|x| *x = rng.gen());
        }

        let can_grow = self.data.len() + 1 < Org::SIZE_LIMIT;

        match rng.gen_range(0..3) {
            0 if can_grow => self.data.push(rng.gen()),
            1 => drop(self.data.pop()),
            _ => {}
        }
        self.fitness = None;
    }

    fn cross(a: &Org, b: &Org) -> Org {
        let split = rand::random::<usize>() % usize::min(a.data.len(), b.data.len());

        let mut data = Vec::with_capacity(b.data.len());
        unsafe {
            data.set_len(b.data.len());
            ptr::copy_nonoverlapping(a.data.as_ptr(), data.as_mut_ptr(), split);
            ptr::copy_nonoverlapping(
                b.data.as_ptr().add(split),
                data.as_mut_ptr().add(split),
                b.data.len() - split,
            );
        }
        Org {
            data,
            fitness: None,
        }
    }
}
