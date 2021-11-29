const { ethers, run } = require("hardhat");
const fs = require("fs");
const assert = require("assert");
const readline = require('readline');

const PARALLEL = 128;

describe("OneStepProof", function () {
  const deployment = run("deploy", { "tags": "OneStepProofEntry" });
  const root = "./test/proofs/";
  const dir = fs.readdirSync(root);
  for (let file of dir) {
    if (!file.endsWith(".json")) continue;
    it("Should pass " + file + " proofs", async function () {
      await deployment;

      let path = root + file;
      let proofs = JSON.parse(fs.readFileSync(path));

      const osp = await ethers.getContract("OneStepProofEntry");

      const promises = [];
      const isdone = [];
      for (let i = 0; i < proofs.length; i++) {
        process.stdout.write("\rTesting " + file + " proof " + i + "/" + proofs.length + " ");
        const proof = proofs[i];
        isdone.push(false);
        const promise = osp.proveOneStep([...Buffer.from(proof.before, "hex")], [...Buffer.from(proof.proof, "hex")])
          .catch(err => {
            console.error("Error executing proof " + i);
            throw err;
          })
          .then(after => assert.equal(after, "0x" + proof.after, "After state doesn't match after proof " + i))
          .finally(_ => {isdone[i] = true});
        if (promises.length < PARALLEL) {
          promises.push(promise);
        } else {
          const finished = await Promise.race(promises.map((p, k) => p.then(_ => k)));
          promises[finished] = promise;
        }
      }

      let stillWaiting = []
      do {
        const finished = await Promise.race(promises.map((p, k) => p.then(_ => k)));
        if (finished == promises.length - 1) {
          promises.pop()
        } else {
          promises[finished] = promises.pop()
        }
        stillWaiting = [];
        for (let i = 0; i < isdone.length; i++) {
          if (!isdone[i]) {
            stillWaiting.push(i)
          }
        }
        readline.clearLine(process.stdout);
        process.stdout.write("\rTesting " + file + " Waiting for: " + String(stillWaiting.length) + "/" + String(isdone.length));
        if (stillWaiting.length < 10) {
          process.stdout.write(": ")
          for (let i = 0; i < stillWaiting.length; i++) {
            process.stdout.write(String(stillWaiting[i]) + ",");
          }
        }
      } while (stillWaiting.length > 0);
      await Promise.all(promises);
      process.stdout.write("\r");
    });
  }
});
