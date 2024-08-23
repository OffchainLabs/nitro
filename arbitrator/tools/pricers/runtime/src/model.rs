/*// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::trial::{Trial, OP_COUNT};
use arbutil::operator::OperatorCode;
use rand::{rngs::ThreadRng, Rng};

pub const WEIGHT_COUNT: usize = OP_COUNT + 1;
const FIXED_WEIGHT: usize = OP_COUNT;

type Weights = [f64; WEIGHT_COUNT];

#[derive(Clone)]
pub struct Model {
    pub weights: Weights,
}

impl Model {
    pub fn new() -> Model {
        let mut weights = [0.; WEIGHT_COUNT];
        weights[FIXED_WEIGHT] = 0.;
        Self { weights }
    }

    pub fn predict(&self, trial: &Trial) -> f64 {
        let fixed = self.weights[OP_COUNT];
        let mut predict = fixed;
        for (count, weight) in trial.ops.iter().zip(self.weights) {
            predict += *count as f64 * weight;
        }
        predict
    }

    pub fn eval(&self, trial: &Trial) -> f64 {
        let predict = self.predict(trial);
        predict - trial.time.as_nanos() as f64
    }

    pub fn tweak(mut self, trial: &Trial) -> Self {
        let mut rng = ThreadRng::default();

        let mut adjust = |weight: &mut f64, _count: usize| {
            let impact = 0.000001;
            let mut draw = || impact - rng.gen::<f64>() % (2. * impact);
            *weight += draw();
            *weight *= 1.0 + draw();
            if !weight.is_finite() {
                *weight = 0.;
            }
            if *weight < 0. {
                *weight = 0.;
            }
        };

        // only tweak weights that were tested
        for (&count, weight) in trial.ops.iter().zip(self.weights.iter_mut()) {
            if count != 0 {
                adjust(weight, count);
            }
        }
        //adjust(&mut self.weights[OP_COUNT], 1);
        self
    }

    pub fn avg(models: &[Model]) -> Model {
        let mut avg = Model::new();
        for model in models {
            for (a, w) in avg.weights.iter_mut().zip(model.weights) {
                *a += w;
            }
        }
        avg.weights = avg.weights.map(|x| x / models.len() as f64);
        avg
    }

    pub fn print(&self, trial: &Trial) {
        for (seq, &count) in trial.ops.iter().enumerate() {
            if count != 0 {
                let op = OperatorCode::from_seq(seq);
                let weight = self.weights[seq];
                println!("{} {:.5}", op, weight);
            }
        }
    }
}
*/
